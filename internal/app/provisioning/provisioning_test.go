package provisioning_test

import (
	"bytes"
	"context"
	"image/png"
	"testing"
	"time"

	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/infra/xui"
)

type mockXUIClient struct {
	addedClients   []xui.AddClientRequest
	updatedClients map[string]xui.ClientPayload
	resetTraffics  []string
	settings       *xui.AllSetting
}

func newMockXUIClient() *mockXUIClient {
	return &mockXUIClient{
		updatedClients: make(map[string]xui.ClientPayload),
		settings: &xui.AllSetting{
			SubURI:  "https://sub.myvpn.com:8443/sub/",
			SubPath: "/sub/",
		},
	}
}

func (m *mockXUIClient) AddClient(ctx context.Context, req xui.AddClientRequest) error {
	m.addedClients = append(m.addedClients, req)
	return nil
}

func (m *mockXUIClient) UpdateClient(ctx context.Context, email string, client xui.ClientPayload) error {
	m.updatedClients[email] = client
	return nil
}

func (m *mockXUIClient) DeleteClient(ctx context.Context, email string) error {
	return nil
}

func (m *mockXUIClient) GetClient(ctx context.Context, email string) (*xui.ClientResponse, error) {
	return nil, nil
}

func (m *mockXUIClient) ListInbounds(ctx context.Context) ([]xui.Inbound, error) {
	return nil, nil
}

func (m *mockXUIClient) GetSubLinks(ctx context.Context, subId string) ([]string, error) {
	return nil, nil
}

func (m *mockXUIClient) ResetTraffic(ctx context.Context, email string) error {
	m.resetTraffics = append(m.resetTraffics, email)
	return nil
}

func (m *mockXUIClient) GetSettings(ctx context.Context) (*xui.AllSetting, error) {
	return m.settings, nil
}

func TestProvisionNewSubscription(t *testing.T) {
	xuiMock := newMockXUIClient()
	svc := provisioning.NewService(xuiMock)
	ctx := context.Background()

	req := provisioning.ProvisionRequest{
		InstanceID:     1,
		UserID:         10,
		TelegramID:     123456789,
		PlanID:         1,
		GroupName:      "vip_group",
		InboundIDs:     []int{1, 2},
		TotalBytes:     53687091200,
		LimitIP:        2,
		DurationMonths: 1,
		PanelURL:       "https://panel.example.com:2053",
	}

	service, qrBytes, err := svc.ProvisionNewSubscription(ctx, req)
	if err != nil {
		t.Fatalf("ProvisionNewSubscription failed: %v", err)
	}

	if len(xuiMock.addedClients) != 1 {
		t.Fatalf("expected 1 client added, got %d", len(xuiMock.addedClients))
	}
	added := xuiMock.addedClients[0].Client
	if added.LimitIP != 2 || added.TotalGB != req.TotalBytes {
		t.Errorf("mismatched client payload: %+v", added)
	}

	if service.SubID == "" || service.ClientUUID == "" || service.ClientEmail == "" {
		t.Errorf("expected auto-generated fields, got %+v", service)
	}

	// Validate PNG format of QR code
	if _, err := png.Decode(bytes.NewReader(qrBytes)); err != nil {
		t.Fatalf("failed to decode generated QR code PNG: %v", err)
	}
}

func TestRenewSubscription(t *testing.T) {
	xuiMock := newMockXUIClient()
	svc := provisioning.NewService(xuiMock)
	ctx := context.Background()

	now := time.Now()
	// Service expiring in 5 days
	initialExpiry := now.Add(5 * 24 * time.Hour)
	service := domain.Service{
		InstanceID:  1,
		ClientEmail: "user_renew@example.com",
		ClientUUID:  "uuid-123",
		SubID:       "sub_existing",
		TotalBytes:  53687091200,
		LimitIP:     2,
		ExpiryTime:  initialExpiry,
		Status:      domain.ServiceStatusActive,
	}

	// Renew for 1 month (30 days)
	renewed, err := svc.RenewSubscription(ctx, service, 1)
	if err != nil {
		t.Fatalf("RenewSubscription failed: %v", err)
	}

	// Stacking verification: new expiry should be initialExpiry + 30 days
	expectedExpiry := initialExpiry.Add(30 * 24 * time.Hour)
	if renewed.ExpiryTime.Sub(expectedExpiry).Abs() > time.Second {
		t.Errorf("expected stacked expiry around %v, got %v", expectedExpiry, renewed.ExpiryTime)
	}

	// Verify traffic was reset
	if len(xuiMock.resetTraffics) != 1 || xuiMock.resetTraffics[0] != service.ClientEmail {
		t.Errorf("traffic reset was not called for %s", service.ClientEmail)
	}

	// Verify client update was sent to 3x-ui
	updated, ok := xuiMock.updatedClients[service.ClientEmail]
	if !ok || updated.ExpiryTime != renewed.ExpiryTimeUnixMs() {
		t.Errorf("updated client mismatch in 3x-ui: %+v", updated)
	}
}

func TestChangeUserCount(t *testing.T) {
	xuiMock := newMockXUIClient()
	svc := provisioning.NewService(xuiMock)
	ctx := context.Background()

	panelURL := "https://panel.example.com:2053"
	service := domain.Service{
		InstanceID:  1,
		ClientEmail: "user_device@example.com",
		ClientUUID:  "uuid-dev",
		SubID:       "sub_initial_123",
		TotalBytes:  53687091200,
		LimitIP:     3,
		ExpiryTime:  time.Now().Add(10 * 24 * time.Hour),
		Status:      domain.ServiceStatusActive,
	}

	// 1. User count increase: 3 -> 4 (no rotation, no new QR code)
	updated, rotated, qr, err := svc.ChangeUserCount(ctx, service, 4, panelURL)
	if err != nil {
		t.Fatalf("ChangeUserCount increase failed: %v", err)
	}
	if rotated {
		t.Errorf("user count increase should not rotate subId")
	}
	if qr != nil {
		t.Errorf("user count increase should not return new QR code")
	}
	if updated.SubID != "sub_initial_123" {
		t.Errorf("subId should remain identical on increase, got %s", updated.SubID)
	}
	if updated.LimitIP != 4 {
		t.Errorf("expected limitIp 4, got %d", updated.LimitIP)
	}

	// 2. User count decrease: 4 -> 2 (P05 & Contract Section 5: MUST rotate subId and issue new QR code!)
	updated2, rotated2, qr2, err := svc.ChangeUserCount(ctx, updated, 2, panelURL)
	if err != nil {
		t.Fatalf("ChangeUserCount decrease failed: %v", err)
	}
	if !rotated2 {
		t.Errorf("user count decrease MUST rotate subId")
	}
	if qr2 == nil {
		t.Errorf("user count decrease MUST return fresh QR code")
	}
	if updated2.SubID == "sub_initial_123" {
		t.Errorf("subId MUST be rotated to a new value")
	}
	if updated2.LimitIP != 2 {
		t.Errorf("expected limitIp 2, got %d", updated2.LimitIP)
	}

	// Verify new QR code is valid PNG
	if _, err := png.Decode(bytes.NewReader(qr2)); err != nil {
		t.Fatalf("failed to decode rotated QR code: %v", err)
	}
}

func TestRotateSubId(t *testing.T) {
	xuiMock := newMockXUIClient()
	svc := provisioning.NewService(xuiMock)
	ctx := context.Background()

	panelURL := "https://panel.example.com:2053"
	service := domain.Service{
		ClientEmail: "user_rot@example.com",
		SubID:       "old_sub_id",
		LimitIP:     2,
	}

	rotatedSvc, qrBytes, err := svc.RotateSubId(ctx, service, panelURL)
	if err != nil {
		t.Fatalf("RotateSubId failed: %v", err)
	}
	if rotatedSvc.SubID == "old_sub_id" || rotatedSvc.SubID == "" {
		t.Errorf("subId not rotated, got %s", rotatedSvc.SubID)
	}
	if qrBytes == nil {
		t.Errorf("expected non-nil QR code")
	}
}
