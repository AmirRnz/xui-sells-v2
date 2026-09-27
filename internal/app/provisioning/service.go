package provisioning

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/infra/xui"
)

var (
	ErrInvalidUserCount = errors.New("new user count must be at least 1")
)

// XUIClient abstracts 3x-ui panel interactions.
type XUIClient interface {
	AddClient(ctx context.Context, req xui.AddClientRequest) error
	UpdateClient(ctx context.Context, email string, client xui.ClientPayload) error
	DeleteClient(ctx context.Context, email string) error
	GetClient(ctx context.Context, email string) (*xui.ClientResponse, error)
	ListInbounds(ctx context.Context) ([]xui.Inbound, error)
	GetSubLinks(ctx context.Context, subId string) ([]string, error)
	ResetTraffic(ctx context.Context, email string) error
	GetSettings(ctx context.Context) (*xui.AllSetting, error)
}

// ProvisionRequest contains parameters to provision a new client in 3x-ui.
type ProvisionRequest struct {
	InstanceID     int64
	UserID         int64
	TelegramID     int64
	PlanID         int64
	ClientEmail    string
	ClientUUID     string
	SubID          string
	GroupName      string
	InboundIDs     []int
	TotalBytes     int64
	LimitIP        int
	DurationMonths int
	PanelURL       string
}

// Service manages client provisioning, renewal, and subscription link rotation.
type Service struct {
	xuiClient XUIClient
}

// NewService creates a new provisioning service.
func NewService(xuiClient XUIClient) *Service {
	return &Service{xuiClient: xuiClient}
}

// GenerateSubID generates a secure random 16-character hex string for subscription links.
func GenerateSubID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ProvisionNewSubscription provisions a client on 3x-ui and generates its QR code.
func (s *Service) ProvisionNewSubscription(ctx context.Context, req ProvisionRequest) (domain.Service, []byte, error) {
	subID := req.SubID
	if subID == "" {
		subID = GenerateSubID()
	}

	clientUUID := req.ClientUUID
	if clientUUID == "" {
		clientUUID = uuid.NewString()
	}

	clientEmail := req.ClientEmail
	if clientEmail == "" {
		clientEmail = fmt.Sprintf("user%d_%s", req.TelegramID, subID[:8])
	}

	var expiryTime time.Time
	var expiryTimeMs int64
	if req.DurationMonths > 0 {
		expiryTime = time.Now().Add(domain.Duration(req.DurationMonths).TimeDuration())
		expiryTimeMs = expiryTime.UnixMilli()
	}

	payload := xui.ClientPayload{
		ID:         clientUUID,
		Email:      clientEmail,
		SubID:      subID,
		TotalGB:    req.TotalBytes,
		ExpiryTime: expiryTimeMs,
		LimitIP:    req.LimitIP,
		TgID:       req.TelegramID,
		Group:      req.GroupName,
		Enable:     true,
	}

	addReq := xui.AddClientRequest{
		Client:     payload,
		InboundIDs: req.InboundIDs,
	}

	if err := s.xuiClient.AddClient(ctx, addReq); err != nil {
		return domain.Service{}, nil, fmt.Errorf("failed to add client to 3x-ui: %w", err)
	}

	subURL, err := s.buildURL(ctx, req.PanelURL, subID)
	if err != nil {
		return domain.Service{}, nil, err
	}

	qrBytes, err := qrcode.Encode(subURL, qrcode.Medium, 256)
	if err != nil {
		return domain.Service{}, nil, fmt.Errorf("failed to generate qr code: %w", err)
	}

	serviceRecord := domain.Service{
		InstanceID:  req.InstanceID,
		UserID:      req.UserID,
		TelegramID:  req.TelegramID,
		PlanID:      req.PlanID,
		ClientEmail: clientEmail,
		ClientUUID:  clientUUID,
		SubID:       subID,
		GroupName:   req.GroupName,
		InboundIDs:  req.InboundIDs,
		TotalBytes:  req.TotalBytes,
		LimitIP:     req.LimitIP,
		ExpiryTime:  expiryTime,
		Status:      domain.ServiceStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return serviceRecord, qrBytes, nil
}

// RenewSubscription extends the subscription expiry and resets traffic (D03).
// If renewed before expiration, new duration stacks onto current expiry.
// If renewed after expiration, duration begins from now.
func (s *Service) RenewSubscription(ctx context.Context, service domain.Service, durationMonths int) (domain.Service, error) {
	duration := domain.Duration(durationMonths)

	var newExpiry time.Time
	if !service.ExpiryTime.IsZero() && service.ExpiryTime.After(time.Now()) {
		newExpiry = service.ExpiryTime.Add(duration.TimeDuration())
	} else {
		newExpiry = time.Now().Add(duration.TimeDuration())
	}

	// 1. Reset client traffic in 3x-ui
	if err := s.xuiClient.ResetTraffic(ctx, service.ClientEmail); err != nil {
		return domain.Service{}, fmt.Errorf("failed to reset client traffic in 3x-ui: %w", err)
	}

	// 2. Update client with new expiry in 3x-ui
	payload := xui.ClientPayload{
		ID:         service.ClientUUID,
		Email:      service.ClientEmail,
		SubID:      service.SubID,
		TotalGB:    service.TotalBytes,
		ExpiryTime: newExpiry.UnixMilli(),
		LimitIP:    service.LimitIP,
		TgID:       service.TelegramID,
		Group:      service.GroupName,
		Enable:     true,
	}

	if err := s.xuiClient.UpdateClient(ctx, service.ClientEmail, payload); err != nil {
		return domain.Service{}, fmt.Errorf("failed to update client expiry in 3x-ui: %w", err)
	}

	service.ExpiryTime = newExpiry
	service.Status = domain.ServiceStatusActive
	service.UpdatedAt = time.Now()

	return service, nil
}

// ChangeUserCount updates the user count (limitIp) in 3x-ui.
// Per P05 & Section 5 of docs/3xui_contract.md:
// - If user count decreased: ROTATES subId, updates client with new limitIp and subId, and generates new QR code.
// - If user count increased: updates client with new limitIp, returns rotated=false and nil QR code.
func (s *Service) ChangeUserCount(ctx context.Context, service domain.Service, newUserCount int, panelURL string) (domain.Service, bool, []byte, error) {
	if newUserCount < 1 {
		return domain.Service{}, false, nil, ErrInvalidUserCount
	}

	if newUserCount == service.LimitIP {
		return service, false, nil, nil
	}

	// User count increased: keep existing subId, update limitIp
	if newUserCount > service.LimitIP {
		payload := xui.ClientPayload{
			ID:         service.ClientUUID,
			Email:      service.ClientEmail,
			SubID:      service.SubID,
			TotalGB:    service.TotalBytes,
			ExpiryTime: service.ExpiryTimeUnixMs(),
			LimitIP:    newUserCount,
			TgID:       service.TelegramID,
			Group:      service.GroupName,
			Enable:     true,
		}

		if err := s.xuiClient.UpdateClient(ctx, service.ClientEmail, payload); err != nil {
			return domain.Service{}, false, nil, fmt.Errorf("failed to update user count in 3x-ui: %w", err)
		}

		service.LimitIP = newUserCount
		service.UpdatedAt = time.Now()
		return service, false, nil, nil
	}

	// User count decreased: ROTATE subId to invalidate old subscription URL (P05)
	newSubID := GenerateSubID()

	payload := xui.ClientPayload{
		ID:         service.ClientUUID,
		Email:      service.ClientEmail,
		SubID:      newSubID,
		TotalGB:    service.TotalBytes,
		ExpiryTime: service.ExpiryTimeUnixMs(),
		LimitIP:    newUserCount,
		TgID:       service.TelegramID,
		Group:      service.GroupName,
		Enable:     true,
	}

	if err := s.xuiClient.UpdateClient(ctx, service.ClientEmail, payload); err != nil {
		return domain.Service{}, false, nil, fmt.Errorf("failed to rotate subId and update user count in 3x-ui: %w", err)
	}

	subURL, err := s.buildURL(ctx, panelURL, newSubID)
	if err != nil {
		return domain.Service{}, false, nil, err
	}

	qrBytes, err := qrcode.Encode(subURL, qrcode.Medium, 256)
	if err != nil {
		return domain.Service{}, false, nil, fmt.Errorf("failed to generate new qr code: %w", err)
	}

	service.SubID = newSubID
	service.LimitIP = newUserCount
	service.UpdatedAt = time.Now()

	return service, true, qrBytes, nil
}

// RotateSubId regenerates a subscription ID, updates 3x-ui, and issues a new QR code.
func (s *Service) RotateSubId(ctx context.Context, service domain.Service, panelURL string) (domain.Service, []byte, error) {
	newSubID := GenerateSubID()

	payload := xui.ClientPayload{
		ID:         service.ClientUUID,
		Email:      service.ClientEmail,
		SubID:      newSubID,
		TotalGB:    service.TotalBytes,
		ExpiryTime: service.ExpiryTimeUnixMs(),
		LimitIP:    service.LimitIP,
		TgID:       service.TelegramID,
		Group:      service.GroupName,
		Enable:     true,
	}

	if err := s.xuiClient.UpdateClient(ctx, service.ClientEmail, payload); err != nil {
		return domain.Service{}, nil, fmt.Errorf("failed to update subId in 3x-ui: %w", err)
	}

	subURL, err := s.buildURL(ctx, panelURL, newSubID)
	if err != nil {
		return domain.Service{}, nil, err
	}

	qrBytes, err := qrcode.Encode(subURL, qrcode.Medium, 256)
	if err != nil {
		return domain.Service{}, nil, fmt.Errorf("failed to generate qr code: %w", err)
	}

	service.SubID = newSubID
	service.UpdatedAt = time.Now()

	return service, qrBytes, nil
}

func (s *Service) buildURL(ctx context.Context, panelURL, subID string) (string, error) {
	settings, err := s.xuiClient.GetSettings(ctx)
	subURI := ""
	subPath := "/sub/"
	if err == nil && settings != nil {
		subURI = settings.SubURI
		subPath = settings.SubPath
	}
	return xui.BuildSubscriptionURL(panelURL, subURI, subPath, subID), nil
}
