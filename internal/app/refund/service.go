package refund

import (
	"context"
	"errors"
	"fmt"
	"time"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

var (
	ErrRefundNotFound         = errors.New("refund request not found")
	ErrRefundWindowExpired    = errors.New("refund eligibility window has expired")
	ErrRefundAlreadyRequested = errors.New("refund has already been requested for this service")
	ErrInvalidRefundStatus    = errors.New("refund request is not pending")
	ErrServiceNotFound        = errors.New("service not found")
	ErrOrderNotFound          = errors.New("order not found")
)

// RefundRepository abstracts persistence for refund requests.
type RefundRepository interface {
	CreateRefundRequest(ctx context.Context, req domain.RefundRequest) (*domain.RefundRequest, error)
	GetRefundRequest(ctx context.Context, id int64) (*domain.RefundRequest, error)
	GetByOrderID(ctx context.Context, orderID int64) (*domain.RefundRequest, error)
	ListPendingRefunds(ctx context.Context, instanceID int64) ([]domain.RefundRequest, error)
	UpdateRefundStatus(ctx context.Context, id int64, status domain.RefundStatus, rejectionReason string) error
}

// OrderRepository abstracts order fetching.
type OrderRepository interface {
	GetOrder(ctx context.Context, orderID int64) (*domain.Order, error)
}

// ServiceRepository abstracts service state changes.
type ServiceRepository interface {
	GetService(ctx context.Context, serviceID int64) (*domain.Service, error)
	UpdateServiceStatus(ctx context.Context, serviceID int64, status domain.ServiceStatus) error
}

// WalletService abstracts balance ledger credits and debits.
type WalletService interface {
	Credit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error)
	Debit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error)
}

// XUIClient abstracts client deletion in 3x-ui upon revocation.
type XUIClient interface {
	DeleteClient(ctx context.Context, email string) error
}

// ReferralClawbackInfo holds referrer details for commission clawback on refund (D04).
type ReferralClawbackInfo struct {
	ReferrerUserID int64
	ReferrerTgID   int64
	Amount         money.Money
}

// Service coordinates customer refund requests and administrator settlements (P14 & D04).
type Service struct {
	refundRepo  RefundRepository
	orderRepo   OrderRepository
	serviceRepo ServiceRepository
	walletSvc   WalletService
	xuiClient   XUIClient
}

// NewService creates a new refund service.
func NewService(
	refundRepo RefundRepository,
	orderRepo OrderRepository,
	serviceRepo ServiceRepository,
	walletSvc WalletService,
	xuiClient XUIClient,
) *Service {
	return &Service{
		refundRepo:  refundRepo,
		orderRepo:   orderRepo,
		serviceRepo: serviceRepo,
		walletSvc:   walletSvc,
		xuiClient:   xuiClient,
	}
}

// RequestRefund creates a pending refund request if within the allowable window (P14).
func (s *Service) RequestRefund(ctx context.Context, instanceID, userID, tgID, serviceID, orderID int64, reason, attachmentPath string, refundWindowDays int) (*domain.RefundRequest, error) {
	order, err := s.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}

	// Verify order was purchased within refund_window_days
	windowDuration := time.Duration(refundWindowDays) * 24 * time.Hour
	if time.Since(order.CreatedAt) > windowDuration {
		return nil, fmt.Errorf("%w: must be requested within %d days", ErrRefundWindowExpired, refundWindowDays)
	}

	// Verify no pending or approved refund already exists for this order
	existing, err := s.refundRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if existing != nil && (existing.Status == domain.RefundStatusPending || existing.Status == domain.RefundStatusApproved) {
		return nil, ErrRefundAlreadyRequested
	}

	req := domain.RefundRequest{
		InstanceID:     instanceID,
		UserID:         userID,
		TelegramID:     tgID,
		ServiceID:      serviceID,
		OrderID:        orderID,
		Amount:         order.Amount,
		Reason:         reason,
		AttachmentPath: attachmentPath,
		Status:         domain.RefundStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return s.refundRepo.CreateRefundRequest(ctx, req)
}

// ListPendingRefunds retrieves all pending refunds for review (P12).
func (s *Service) ListPendingRefunds(ctx context.Context, instanceID int64) ([]domain.RefundRequest, error) {
	return s.refundRepo.ListPendingRefunds(ctx, instanceID)
}

// ApproveRefund approves the refund request: credits user wallet, revokes 3x-ui client, and claws back referral commission (D04).
func (s *Service) ApproveRefund(ctx context.Context, refundID int64, clawback *ReferralClawbackInfo) error {
	req, err := s.refundRepo.GetRefundRequest(ctx, refundID)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrRefundNotFound
	}

	// Idempotent: return nil if already approved
	if req.Status == domain.RefundStatusApproved {
		return nil
	}
	if req.Status != domain.RefundStatusPending {
		return fmt.Errorf("%w: status %s", ErrInvalidRefundStatus, req.Status)
	}

	// 1. Refund purchase price to customer's wallet balance
	if s.walletSvc != nil {
		refID := fmt.Sprintf("refund_%d", req.ID)
		desc := fmt.Sprintf("Refund for order #%d", req.OrderID)
		_, err := s.walletSvc.Credit(ctx, req.InstanceID, req.UserID, req.TelegramID, req.Amount, domain.TxTypeRefund, refID, desc)
		if err != nil {
			return fmt.Errorf("failed to credit customer wallet for refund: %w", err)
		}
	}

	// 2. Referral commission clawback if applicable (D04)
	if clawback != nil && clawback.Amount.IsPositive() && s.walletSvc != nil {
		refID := fmt.Sprintf("refund_%d", req.ID)
		desc := fmt.Sprintf("Referral commission clawback for refunded order #%d", req.OrderID)
		_, err := s.walletSvc.Debit(ctx, req.InstanceID, clawback.ReferrerUserID, clawback.ReferrerTgID, clawback.Amount, domain.TxTypeReferralClawback, refID, desc)
		if err != nil {
			// Non-blocking log if referrer has insufficient balance, or proceed
		}
	}

	// 3. Service Revocation: disable/delete in 3x-ui and mark status revoked
	if s.serviceRepo != nil {
		svc, err := s.serviceRepo.GetService(ctx, req.ServiceID)
		if err == nil && svc != nil {
			if s.xuiClient != nil {
				_ = s.xuiClient.DeleteClient(ctx, svc.ClientEmail)
			}
			_ = s.serviceRepo.UpdateServiceStatus(ctx, req.ServiceID, domain.ServiceStatusRevoked)
		}
	}

	// 4. Mark refund approved
	return s.refundRepo.UpdateRefundStatus(ctx, refundID, domain.RefundStatusApproved, "")
}

// RejectRefund rejects the refund request with a recorded explanation.
func (s *Service) RejectRefund(ctx context.Context, refundID int64, reason string) error {
	req, err := s.refundRepo.GetRefundRequest(ctx, refundID)
	if err != nil {
		return err
	}
	if req == nil {
		return ErrRefundNotFound
	}

	if req.Status == domain.RefundStatusRejected {
		return nil
	}
	if req.Status != domain.RefundStatusPending {
		return fmt.Errorf("%w: status %s", ErrInvalidRefundStatus, req.Status)
	}

	return s.refundRepo.UpdateRefundStatus(ctx, refundID, domain.RefundStatusRejected, reason)
}
