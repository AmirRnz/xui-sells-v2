package order_test

import (
	"context"
	"testing"
	"time"

	"xui-sells-v2/internal/app/order"
	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

type mockOrderRepo struct {
	orders map[int64]*domain.Order
	nextID int64
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders: make(map[int64]*domain.Order),
		nextID: 1,
	}
}

func (m *mockOrderRepo) CreateOrder(ctx context.Context, o domain.Order) (*domain.Order, error) {
	o.ID = m.nextID
	m.nextID++
	cp := o
	m.orders[o.ID] = &cp
	return &cp, nil
}

func (m *mockOrderRepo) GetOrder(ctx context.Context, orderID int64) (*domain.Order, error) {
	o, ok := m.orders[orderID]
	if !ok {
		return nil, nil
	}
	cp := *o
	return &cp, nil
}

func (m *mockOrderRepo) UpdateOrderStatus(ctx context.Context, orderID int64, status domain.OrderStatus, reason string) error {
	o, ok := m.orders[orderID]
	if !ok {
		return order.ErrOrderNotFound
	}
	o.Status = status
	o.RejectionReason = reason
	o.UpdatedAt = time.Now()
	return nil
}

type mockServiceRepo struct {
	services map[int64]*domain.Service
	nextID   int64
}

func newMockServiceRepo() *mockServiceRepo {
	return &mockServiceRepo{
		services: make(map[int64]*domain.Service),
		nextID:   1,
	}
}

func (m *mockServiceRepo) CreateService(ctx context.Context, s domain.Service) (*domain.Service, error) {
	s.ID = m.nextID
	m.nextID++
	cp := s
	m.services[s.ID] = &cp
	return &cp, nil
}

func (m *mockServiceRepo) GetService(ctx context.Context, serviceID int64) (*domain.Service, error) {
	s, ok := m.services[serviceID]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (m *mockServiceRepo) UpdateService(ctx context.Context, s domain.Service) error {
	cp := s
	m.services[s.ID] = &cp
	return nil
}

type mockWalletSvc struct {
	credits            []domain.WalletTransaction
	debits             []domain.WalletTransaction
	settledReservations []int64
	releasedReservations []int64
}

func (m *mockWalletSvc) Credit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error) {
	tx := domain.WalletTransaction{
		InstanceID:  instanceID,
		UserID:      userID,
		TelegramID:  tgID,
		Type:        txType,
		Amount:      amount,
		ReferenceID: refID,
		Description: desc,
	}
	m.credits = append(m.credits, tx)
	return &tx, nil
}

func (m *mockWalletSvc) Debit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error) {
	tx := domain.WalletTransaction{
		InstanceID:  instanceID,
		UserID:      userID,
		TelegramID:  tgID,
		Type:        txType,
		Amount:      amount,
		ReferenceID: refID,
		Description: desc,
	}
	m.debits = append(m.debits, tx)
	return &tx, nil
}

func (m *mockWalletSvc) SettleReservation(ctx context.Context, resID int64, resellerUserID int64) error {
	m.settledReservations = append(m.settledReservations, resID)
	return nil
}

func (m *mockWalletSvc) ReleaseReservation(ctx context.Context, resID int64) error {
	m.releasedReservations = append(m.releasedReservations, resID)
	return nil
}

type mockProvSvc struct {
	provisionCalls int
	renewCalls     int
	userCountCalls int
}

func (m *mockProvSvc) ProvisionNewSubscription(ctx context.Context, req provisioning.ProvisionRequest) (domain.Service, []byte, error) {
	m.provisionCalls++
	return domain.Service{
		ClientEmail: "new_client@vpn.com",
		SubID:       "sub_test",
	}, []byte("fake_qr_png"), nil
}

func (m *mockProvSvc) RenewSubscription(ctx context.Context, s domain.Service, duration int) (domain.Service, error) {
	m.renewCalls++
	s.ExpiryTime = time.Now().Add(30 * 24 * time.Hour)
	return s, nil
}

func (m *mockProvSvc) ChangeUserCount(ctx context.Context, s domain.Service, count int, url string) (domain.Service, bool, []byte, error) {
	m.userCountCalls++
	s.LimitIP = count
	return s, false, nil, nil
}

func TestApproveOrderWithProvisioningAndReferral(t *testing.T) {
	orderRepo := newMockOrderRepo()
	serviceRepo := newMockServiceRepo()
	walletSvc := &mockWalletSvc{}
	provSvc := &mockProvSvc{}

	svc := order.NewService(orderRepo, serviceRepo, walletSvc, provSvc)
	ctx := context.Background()

	// Create pending purchase order for 200,000 Toman
	newOrder, err := svc.CreateOrder(ctx, domain.Order{
		InstanceID:    1,
		UserID:        10,
		TelegramID:    12345,
		Type:          domain.OrderTypePurchase,
		Amount:        money.NewToman(200_000),
		PaymentMethod: domain.PaymentMethodDirect,
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	reservationID := int64(77)
	resellerUserID := int64(99)
	referral := &order.ReferralInfo{
		ReferrerUserID: 50,
		ReferrerTgID:   8888,
		Percentage:     10, // 10% of 200,000 = 20,000 Toman
	}

	opts := order.ApprovalOptions{
		ReservationID:  &reservationID,
		ResellerUserID: &resellerUserID,
		Referral:       referral,
		ProvisionReq: &provisioning.ProvisionRequest{
			TelegramID: 12345,
			TotalBytes: 53687091200,
		},
	}

	// Approve order
	if err := svc.ApproveOrder(ctx, newOrder.ID, opts); err != nil {
		t.Fatalf("ApproveOrder failed: %v", err)
	}

	// Check provisioning called
	if provSvc.provisionCalls != 1 {
		t.Errorf("expected 1 provisioning call, got %d", provSvc.provisionCalls)
	}

	// Check reservation settled
	if len(walletSvc.settledReservations) != 1 || walletSvc.settledReservations[0] != 77 {
		t.Errorf("expected reservation 77 settled, got %v", walletSvc.settledReservations)
	}

	// Check referral reward credited: 10% of 200,000 is 20,000
	if len(walletSvc.credits) != 1 {
		t.Fatalf("expected 1 referral credit, got %d", len(walletSvc.credits))
	}
	refCredit := walletSvc.credits[0]
	if refCredit.UserID != 50 || refCredit.Amount.Amount != 20_000 || refCredit.Type != domain.TxTypeReferralReward {
		t.Errorf("unexpected referral credit: %+v", refCredit)
	}

	// IDEMPOTENCY: second approval must safely return nil without duplicate provisioning or credits
	if err := svc.ApproveOrder(ctx, newOrder.ID, opts); err != nil {
		t.Fatalf("idempotent approval failed: %v", err)
	}
	if provSvc.provisionCalls != 1 {
		t.Errorf("provisioning called multiple times on duplicate approval!")
	}
	if len(walletSvc.credits) != 1 {
		t.Errorf("duplicate referral credit on duplicate approval!")
	}
}

func TestRejectOrderWithWalletRefundAndReservationRelease(t *testing.T) {
	orderRepo := newMockOrderRepo()
	serviceRepo := newMockServiceRepo()
	walletSvc := &mockWalletSvc{}
	provSvc := &mockProvSvc{}

	svc := order.NewService(orderRepo, serviceRepo, walletSvc, provSvc)
	ctx := context.Background()

	// Create pending wallet-paid order
	newOrder, err := svc.CreateOrder(ctx, domain.Order{
		InstanceID:    1,
		UserID:        10,
		TelegramID:    12345,
		Type:          domain.OrderTypePurchase,
		Amount:        money.NewToman(150_000),
		PaymentMethod: domain.PaymentMethodWallet,
	})
	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	resID := int64(88)
	rejectOpts := order.RejectionOptions{
		Reason:        "Out of stock",
		ReservationID: &resID,
	}

	// Reject order
	if err := svc.RejectOrder(ctx, newOrder.ID, rejectOpts); err != nil {
		t.Fatalf("RejectOrder failed: %v", err)
	}

	// Verify wallet was refunded for the customer
	if len(walletSvc.credits) != 1 {
		t.Fatalf("expected 1 refund credit, got %d", len(walletSvc.credits))
	}
	refund := walletSvc.credits[0]
	if refund.UserID != 10 || refund.Amount.Amount != 150_000 || refund.Type != domain.TxTypeRefund {
		t.Errorf("unexpected wallet refund: %+v", refund)
	}

	// Verify reseller reservation was released
	if len(walletSvc.releasedReservations) != 1 || walletSvc.releasedReservations[0] != 88 {
		t.Errorf("expected reservation 88 released, got %v", walletSvc.releasedReservations)
	}

	// IDEMPOTENCY: second rejection call must return nil without duplicate refunds
	if err := svc.RejectOrder(ctx, newOrder.ID, rejectOpts); err != nil {
		t.Fatalf("idempotent rejection failed: %v", err)
	}
	if len(walletSvc.credits) != 1 {
		t.Errorf("duplicate wallet refund on duplicate rejection!")
	}
}
