package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidOrderStatus = errors.New("order is not in a pending status")
)

// OrderRepository abstracts order persistence.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order domain.Order) (*domain.Order, error)
	GetOrder(ctx context.Context, orderID int64) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status domain.OrderStatus, reason string) error
}

// WalletService abstracts wallet operations required during order lifecycle.
type WalletService interface {
	Credit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error)
	Debit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error)
	SettleReservation(ctx context.Context, reservationID int64, resellerUserID int64) error
	ReleaseReservation(ctx context.Context, reservationID int64) error
}

// ProvisioningService abstracts subscription operations in 3x-ui.
type ProvisioningService interface {
	ProvisionNewSubscription(ctx context.Context, req provisioning.ProvisionRequest) (domain.Service, []byte, error)
	RenewSubscription(ctx context.Context, service domain.Service, durationMonths int) (domain.Service, error)
	ChangeUserCount(ctx context.Context, service domain.Service, newUserCount int, panelURL string) (domain.Service, bool, []byte, error)
}

// ServiceRepository abstracts persistence for active VPN services.
type ServiceRepository interface {
	CreateService(ctx context.Context, svc domain.Service) (*domain.Service, error)
	GetService(ctx context.Context, serviceID int64) (*domain.Service, error)
	UpdateService(ctx context.Context, svc domain.Service) error
}

// ReferralInfo holds referrer data to reward commission upon order approval.
type ReferralInfo struct {
	ReferrerUserID int64
	ReferrerTgID   int64
	Percentage     int
}

// ApprovalOptions holds contextual references for order fulfillment.
type ApprovalOptions struct {
	ReservationID  *int64
	ResellerUserID *int64
	Referral       *ReferralInfo
	ProvisionReq   *provisioning.ProvisionRequest
	PanelURL       string
}

// RejectionOptions holds references needed when rejecting an order.
type RejectionOptions struct {
	Reason        string
	ReservationID *int64
}

// Service coordinates the order state machine, payments, provisioning, and referral rewards.
type Service struct {
	orderRepo   OrderRepository
	serviceRepo ServiceRepository
	walletSvc   WalletService
	provSvc     ProvisioningService
}

// NewService creates a new order service.
func NewService(
	orderRepo OrderRepository,
	serviceRepo ServiceRepository,
	walletSvc WalletService,
	provSvc ProvisioningService,
) *Service {
	return &Service{
		orderRepo:   orderRepo,
		serviceRepo: serviceRepo,
		walletSvc:   walletSvc,
		provSvc:     provSvc,
	}
}

// CreateOrder registers a new pending purchase, renewal, topup, or user count change order.
func (s *Service) CreateOrder(ctx context.Context, order domain.Order) (*domain.Order, error) {
	if order.Status == "" {
		order.Status = domain.OrderStatusPendingApproval
	}
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	return s.orderRepo.CreateOrder(ctx, order)
}

// ApproveOrder idempotently approves an order, triggers provisioning, settles reseller holds,
// and rewards referrers per P04, P15, P20, and D05.
func (s *Service) ApproveOrder(ctx context.Context, orderID int64, opts ApprovalOptions) error {
	order, err := s.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	// Idempotency: if already approved, safely return without re-provisioning
	if order.Status == domain.OrderStatusApproved {
		return nil
	}
	if order.Status != domain.OrderStatusPendingApproval {
		return fmt.Errorf("%w: current status %s", ErrInvalidOrderStatus, order.Status)
	}

	// 1. Provisioning / Fulfillment based on OrderType
	switch order.Type {
	case domain.OrderTypePurchase:
		if opts.ProvisionReq != nil && s.provSvc != nil {
			newSvc, _, err := s.provSvc.ProvisionNewSubscription(ctx, *opts.ProvisionReq)
			if err != nil {
				return fmt.Errorf("failed to provision new subscription: %w", err)
			}
			if s.serviceRepo != nil {
				savedSvc, err := s.serviceRepo.CreateService(ctx, newSvc)
				if err != nil {
					return fmt.Errorf("failed to save provisioned service: %w", err)
				}
				order.ServiceID = &savedSvc.ID
			}
		}

	case domain.OrderTypeRenewal:
		if order.ServiceID != nil && s.serviceRepo != nil && s.provSvc != nil {
			existing, err := s.serviceRepo.GetService(ctx, *order.ServiceID)
			if err != nil {
				return fmt.Errorf("failed to retrieve service for renewal: %w", err)
			}
			if existing != nil {
				renewed, err := s.provSvc.RenewSubscription(ctx, *existing, order.DurationMonths)
				if err != nil {
					return fmt.Errorf("failed to renew subscription in 3x-ui: %w", err)
				}
				if err := s.serviceRepo.UpdateService(ctx, renewed); err != nil {
					return fmt.Errorf("failed to update renewed service: %w", err)
				}
			}
		}

	case domain.OrderTypeUserCountChange:
		if order.ServiceID != nil && s.serviceRepo != nil && s.provSvc != nil {
			existing, err := s.serviceRepo.GetService(ctx, *order.ServiceID)
			if err != nil {
				return fmt.Errorf("failed to retrieve service for user count change: %w", err)
			}
			if existing != nil {
				updatedSvc, _, _, err := s.provSvc.ChangeUserCount(ctx, *existing, order.UserCount, opts.PanelURL)
				if err != nil {
					return fmt.Errorf("failed to change user count in 3x-ui: %w", err)
				}
				if err := s.serviceRepo.UpdateService(ctx, updatedSvc); err != nil {
					return fmt.Errorf("failed to update service after user count change: %w", err)
				}
			}
		}

	case domain.OrderTypeTopup:
		if s.walletSvc != nil {
			refID := fmt.Sprintf("order_%d", order.ID)
			_, err := s.walletSvc.Credit(ctx, order.InstanceID, order.UserID, order.TelegramID, order.Amount, domain.TxTypeTopup, refID, "Wallet topup approved")
			if err != nil {
				return fmt.Errorf("failed to credit wallet on topup approval: %w", err)
			}
		}
	}

	// 2. Settle Reseller Wholesale Reservation if this was a child-bot order (P20)
	if opts.ReservationID != nil && opts.ResellerUserID != nil && s.walletSvc != nil {
		if err := s.walletSvc.SettleReservation(ctx, *opts.ReservationID, *opts.ResellerUserID); err != nil {
			return fmt.Errorf("failed to settle reseller wholesale reservation: %w", err)
		}
	}

	// 3. Referral Commission Settlement (P15 & D05)
	// Anti-Double-Counting rule: wallet topups NEVER grant referral rewards. Only finalized purchases generate rewards.
	if order.Type != domain.OrderTypeTopup && opts.Referral != nil && opts.Referral.Percentage > 0 && opts.Referral.ReferrerUserID > 0 && s.walletSvc != nil {
		rawCommission := order.Amount.Amount * int64(opts.Referral.Percentage)
		commissionAmount := (rawCommission + 50) / 100
		commission := money.New(commissionAmount, order.Amount.Currency)

		refID := fmt.Sprintf("order_%d", order.ID)
		desc := fmt.Sprintf("Referral reward (%d%%) from user #%d order #%d", opts.Referral.Percentage, order.TelegramID, order.ID)

		_, err := s.walletSvc.Credit(ctx, order.InstanceID, opts.Referral.ReferrerUserID, opts.Referral.ReferrerTgID, commission, domain.TxTypeReferralReward, refID, desc)
		if err != nil {
			return fmt.Errorf("failed to credit referral commission: %w", err)
		}
	}

	// 4. Mark order approved
	return s.orderRepo.UpdateOrderStatus(ctx, orderID, domain.OrderStatusApproved, "")
}

// RejectOrder idempotently rejects an order, refunds wallet balance if paid by wallet,
// and releases any reseller reservation hold (P12, P14, P20).
func (s *Service) RejectOrder(ctx context.Context, orderID int64, opts RejectionOptions) error {
	order, err := s.orderRepo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if order == nil {
		return ErrOrderNotFound
	}

	// Idempotency: if already rejected, safely return
	if order.Status == domain.OrderStatusRejected {
		return nil
	}
	if order.Status != domain.OrderStatusPendingApproval {
		return fmt.Errorf("%w: current status %s", ErrInvalidOrderStatus, order.Status)
	}

	// 1. If paid by wallet, refund the customer balance
	if order.PaymentMethod == domain.PaymentMethodWallet && s.walletSvc != nil {
		refID := fmt.Sprintf("order_%d", order.ID)
		desc := fmt.Sprintf("Refund for rejected order #%d: %s", order.ID, opts.Reason)
		_, err := s.walletSvc.Credit(ctx, order.InstanceID, order.UserID, order.TelegramID, order.Amount, domain.TxTypeRefund, refID, desc)
		if err != nil {
			return fmt.Errorf("failed to refund wallet balance on rejection: %w", err)
		}
	}

	// 2. If child-bot wholesale reservation exists, release the hold
	if opts.ReservationID != nil && s.walletSvc != nil {
		if err := s.walletSvc.ReleaseReservation(ctx, *opts.ReservationID); err != nil {
			return fmt.Errorf("failed to release reseller reservation on rejection: %w", err)
		}
	}

	// 3. Mark order rejected
	return s.orderRepo.UpdateOrderStatus(ctx, orderID, domain.OrderStatusRejected, opts.Reason)
}
