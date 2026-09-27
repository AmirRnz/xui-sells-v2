package refund_test

import (
	"context"
	"testing"
	"time"

	"xui-sells-v2/internal/app/refund"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

type mockRefundRepo struct {
	refunds map[int64]*domain.RefundRequest
	nextID  int64
}

func newMockRefundRepo() *mockRefundRepo {
	return &mockRefundRepo{
		refunds: make(map[int64]*domain.RefundRequest),
		nextID:  1,
	}
}

func (m *mockRefundRepo) CreateRefundRequest(ctx context.Context, r domain.RefundRequest) (*domain.RefundRequest, error) {
	r.ID = m.nextID
	m.nextID++
	cp := r
	m.refunds[r.ID] = &cp
	return &cp, nil
}

func (m *mockRefundRepo) GetRefundRequest(ctx context.Context, id int64) (*domain.RefundRequest, error) {
	r, ok := m.refunds[id]
	if !ok {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}

func (m *mockRefundRepo) GetByOrderID(ctx context.Context, orderID int64) (*domain.RefundRequest, error) {
	for _, r := range m.refunds {
		if r.OrderID == orderID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockRefundRepo) ListPendingRefunds(ctx context.Context, instanceID int64) ([]domain.RefundRequest, error) {
	var list []domain.RefundRequest
	for _, r := range m.refunds {
		if r.InstanceID == instanceID && r.Status == domain.RefundStatusPending {
			list = append(list, *r)
		}
	}
	return list, nil
}

func (m *mockRefundRepo) UpdateRefundStatus(ctx context.Context, id int64, status domain.RefundStatus, reason string) error {
	r, ok := m.refunds[id]
	if !ok {
		return refund.ErrRefundNotFound
	}
	r.Status = status
	r.RejectionReason = reason
	return nil
}

type mockOrderRepo struct {
	order *domain.Order
}

func (m *mockOrderRepo) GetOrder(ctx context.Context, orderID int64) (*domain.Order, error) {
	return m.order, nil
}

type mockServiceRepo struct {
	service *domain.Service
}

func (m *mockServiceRepo) GetService(ctx context.Context, id int64) (*domain.Service, error) {
	return m.service, nil
}

func (m *mockServiceRepo) UpdateServiceStatus(ctx context.Context, id int64, status domain.ServiceStatus) error {
	if m.service != nil {
		m.service.Status = status
	}
	return nil
}

type mockWalletSvc struct {
	credits []domain.WalletTransaction
	debits  []domain.WalletTransaction
}

func (m *mockWalletSvc) Credit(ctx context.Context, instanceID, userID, tgID int64, amount money.Money, txType domain.TxType, refID, desc string) (*domain.WalletTransaction, error) {
	tx := domain.WalletTransaction{
		InstanceID: instanceID,
		UserID:     userID,
		TelegramID: tgID,
		Amount:     amount,
		Type:       txType,
	}
	m.credits = append(m.credits, tx)
	return &tx, nil
}

func (m *mockWalletSvc) Debit(ctx context.Context, instanceID, userID, tgID int64, amount money.Money, txType domain.TxType, refID, desc string) (*domain.WalletTransaction, error) {
	tx := domain.WalletTransaction{
		InstanceID: instanceID,
		UserID:     userID,
		TelegramID: tgID,
		Amount:     amount,
		Type:       txType,
	}
	m.debits = append(m.debits, tx)
	return &tx, nil
}

type mockXUI struct {
	deletedEmails []string
}

func (m *mockXUI) DeleteClient(ctx context.Context, email string) error {
	m.deletedEmails = append(m.deletedEmails, email)
	return nil
}

func TestRefundWorkflowAndClawback(t *testing.T) {
	refundRepo := newMockRefundRepo()
	orderRepo := &mockOrderRepo{
		order: &domain.Order{
			ID:        100,
			Amount:    money.NewToman(200_000),
			CreatedAt: time.Now().Add(-12 * time.Hour), // 12 hours ago
		},
	}
	serviceRepo := &mockServiceRepo{
		service: &domain.Service{
			ID:          50,
			ClientEmail: "revoke_me@vpn.com",
			Status:      domain.ServiceStatusActive,
		},
	}
	walletSvc := &mockWalletSvc{}
	xuiMock := &mockXUI{}

	svc := refund.NewService(refundRepo, orderRepo, serviceRepo, walletSvc, xuiMock)
	ctx := context.Background()

	// 1. Request Refund within 2-day window -> Success
	req, err := svc.RequestRefund(ctx, 1, 10, 12345, 50, 100, "Slow speed", "", 2)
	if err != nil {
		t.Fatalf("RequestRefund failed: %v", err)
	}
	if req.Status != domain.RefundStatusPending || req.Amount.Amount != 200_000 {
		t.Errorf("unexpected refund request: %+v", req)
	}

	// 2. Duplicate request -> Error
	_, err = svc.RequestRefund(ctx, 1, 10, 12345, 50, 100, "Slow speed again", "", 2)
	if err != refund.ErrRefundAlreadyRequested {
		t.Errorf("expected ErrRefundAlreadyRequested, got %v", err)
	}

	// 3. Approve Refund with referral clawback (20,000 Toman)
	clawback := &refund.ReferralClawbackInfo{
		ReferrerUserID: 77,
		ReferrerTgID:   7777,
		Amount:         money.NewToman(20_000),
	}

	err = svc.ApproveRefund(ctx, req.ID, clawback)
	if err != nil {
		t.Fatalf("ApproveRefund failed: %v", err)
	}

	// Verify customer wallet credited
	if len(walletSvc.credits) != 1 || walletSvc.credits[0].Amount.Amount != 200_000 {
		t.Errorf("customer wallet not credited properly: %+v", walletSvc.credits)
	}

	// Verify referrer wallet debited (clawback)
	if len(walletSvc.debits) != 1 || walletSvc.debits[0].UserID != 77 || walletSvc.debits[0].Amount.Amount != 20_000 {
		t.Errorf("referrer clawback not recorded: %+v", walletSvc.debits)
	}

	// Verify client deleted from 3x-ui
	if len(xuiMock.deletedEmails) != 1 || xuiMock.deletedEmails[0] != "revoke_me@vpn.com" {
		t.Errorf("service not deleted from 3x-ui")
	}

	// Verify service marked revoked
	if serviceRepo.service.Status != domain.ServiceStatusRevoked {
		t.Errorf("service not marked revoked")
	}

	// IDEMPOTENCY: duplicate approval returns nil
	if err := svc.ApproveRefund(ctx, req.ID, clawback); err != nil {
		t.Fatalf("duplicate approval failed: %v", err)
	}
}

func TestRefundWindowExpired(t *testing.T) {
	refundRepo := newMockRefundRepo()
	orderRepo := &mockOrderRepo{
		order: &domain.Order{
			ID:        101,
			Amount:    money.NewToman(200_000),
			CreatedAt: time.Now().Add(-5 * 24 * time.Hour), // 5 days ago
		},
	}
	svc := refund.NewService(refundRepo, orderRepo, nil, nil, nil)

	// Refund window is 2 days -> should fail with ErrRefundWindowExpired
	_, err := svc.RequestRefund(context.Background(), 1, 10, 12345, 50, 101, "Expired request", "", 2)
	if err == nil {
		t.Fatalf("expected error for expired refund window")
	}
}
