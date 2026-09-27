package wallet_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"xui-sells-v2/internal/app/wallet"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

type inMemoryWalletRepo struct {
	mu           sync.Mutex
	balances     map[string]money.Money
	transactions []domain.WalletTransaction
	reservations map[int64]*domain.ResellerFundingReservation
	nextResID    int64
}

func newInMemoryWalletRepo() *inMemoryWalletRepo {
	return &inMemoryWalletRepo{
		balances:     make(map[string]money.Money),
		reservations: make(map[int64]*domain.ResellerFundingReservation),
		nextResID:    1,
	}
}

func (m *inMemoryWalletRepo) key(instanceID, userID int64) string {
	return fmt.Sprintf("%d:%d", instanceID, userID)
}

func (m *inMemoryWalletRepo) GetBalance(ctx context.Context, instanceID int64, userID int64) (money.Money, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.balances[m.key(instanceID, userID)]
	if !ok {
		return money.NewToman(0), nil
	}
	return b, nil
}

func (m *inMemoryWalletRepo) GetAvailableBalance(ctx context.Context, instanceID int64, userID int64) (money.Money, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.balances[m.key(instanceID, userID)]
	if !ok {
		b = money.NewToman(0)
	}

	// Deduct pending reservations
	pendingTotal := money.New(0, b.Currency)
	for _, res := range m.reservations {
		if res.ParentInstanceID == instanceID && res.Status == domain.ReservationStatusPending {
			pendingTotal, _ = pendingTotal.Add(res.Amount)
		}
	}

	avail, _ := b.Sub(pendingTotal)
	return avail, nil
}

func (m *inMemoryWalletRepo) RecordTransaction(ctx context.Context, tx domain.WalletTransaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balances[m.key(tx.InstanceID, tx.UserID)] = tx.BalanceAfter
	m.transactions = append(m.transactions, tx)
	return nil
}

func (m *inMemoryWalletRepo) CreateReservation(ctx context.Context, res domain.ResellerFundingReservation) (*domain.ResellerFundingReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res.ID = m.nextResID
	m.nextResID++
	copied := res
	m.reservations[res.ID] = &copied
	return &copied, nil
}

func (m *inMemoryWalletRepo) GetReservation(ctx context.Context, reservationID int64) (*domain.ResellerFundingReservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res, ok := m.reservations[reservationID]
	if !ok {
		return nil, nil
	}
	copied := *res
	return &copied, nil
}

func (m *inMemoryWalletRepo) UpdateReservationStatus(ctx context.Context, reservationID int64, status domain.ReservationStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	res, ok := m.reservations[reservationID]
	if !ok {
		return wallet.ErrReservationNotFound
	}
	res.Status = status
	res.UpdatedAt = time.Now()
	return nil
}

func TestCreditAndDebit(t *testing.T) {
	repo := newInMemoryWalletRepo()
	svc := wallet.NewService(repo)
	ctx := context.Background()

	// Initial Credit
	creditAmount := money.NewToman(200_000)
	tx, err := svc.Credit(ctx, 1, 10, 100, creditAmount, domain.TxTypeTopup, "topup_1", "Test deposit")
	if err != nil {
		t.Fatalf("Credit failed: %v", err)
	}
	if tx.BalanceAfter.Amount != 200_000 {
		t.Errorf("expected balance 200,000, got %d", tx.BalanceAfter.Amount)
	}

	// Valid Debit
	debitAmount := money.NewToman(150_000)
	txDebit, err := svc.Debit(ctx, 1, 10, 100, debitAmount, domain.TxTypePurchase, "order_1", "Test purchase")
	if err != nil {
		t.Fatalf("Debit failed: %v", err)
	}
	if txDebit.BalanceAfter.Amount != 50_000 {
		t.Errorf("expected balance 50,000, got %d", txDebit.BalanceAfter.Amount)
	}

	// Overdraft Debit (Should fail with ErrInsufficientFunds)
	overdraft := money.NewToman(100_000)
	_, err = svc.Debit(ctx, 1, 10, 100, overdraft, domain.TxTypePurchase, "order_2", "Overdraft attempt")
	if err != wallet.ErrInsufficientFunds {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestResellerSharedCreditReservationWorkflow(t *testing.T) {
	repo := newInMemoryWalletRepo()
	svc := wallet.NewService(repo)
	ctx := context.Background()

	parentInstanceID := int64(1)
	childInstanceID := int64(2)
	resellerUserID := int64(55)
	resellerTgID := int64(999999)

	// Reseller has 500,000 Toman in parent instance
	_, err := svc.Credit(ctx, parentInstanceID, resellerUserID, resellerTgID, money.NewToman(500_000), domain.TxTypeTopup, "topup_r", "Reseller funding")
	if err != nil {
		t.Fatalf("reseller funding credit failed: %v", err)
	}

	// 1. Reserve 200,000 for child bot order #101
	res, err := svc.ReserveResellerCredit(ctx, parentInstanceID, childInstanceID, resellerTgID, resellerUserID, 101, money.NewToman(200_000))
	if err != nil {
		t.Fatalf("reservation failed: %v", err)
	}
	if res.Status != domain.ReservationStatusPending {
		t.Errorf("expected pending status, got %s", res.Status)
	}

	// Available balance is now 500k - 200k = 300k
	avail, err := repo.GetAvailableBalance(ctx, parentInstanceID, resellerUserID)
	if err != nil || avail.Amount != 300_000 {
		t.Errorf("expected available balance 300,000, got %d (err: %v)", avail.Amount, err)
	}

	// 2. Attempting to reserve 400,000 should fail (only 300k available)
	_, err = svc.ReserveResellerCredit(ctx, parentInstanceID, childInstanceID, resellerTgID, resellerUserID, 102, money.NewToman(400_000))
	if err != wallet.ErrInsufficientResellerFunds {
		t.Errorf("expected ErrInsufficientResellerFunds, got %v", err)
	}

	// 3. Settle reservation #1: actual deduction of 200,000 from wallet
	err = svc.SettleReservation(ctx, res.ID, resellerUserID)
	if err != nil {
		t.Fatalf("settlement failed: %v", err)
	}

	// Raw balance is now 300,000, available is 300,000
	bal, _ := repo.GetBalance(ctx, parentInstanceID, resellerUserID)
	if bal.Amount != 300_000 {
		t.Errorf("expected balance 300,000 after settlement, got %d", bal.Amount)
	}

	// Idempotent settlement call
	if err := svc.SettleReservation(ctx, res.ID, resellerUserID); err != nil {
		t.Fatalf("idempotent settlement failed: %v", err)
	}

	// 4. Reserve and Release
	res2, err := svc.ReserveResellerCredit(ctx, parentInstanceID, childInstanceID, resellerTgID, resellerUserID, 103, money.NewToman(100_000))
	if err != nil {
		t.Fatalf("reservation 2 failed: %v", err)
	}
	err = svc.ReleaseReservation(ctx, res2.ID)
	if err != nil {
		t.Fatalf("release failed: %v", err)
	}

	availAfterRelease, _ := repo.GetAvailableBalance(ctx, parentInstanceID, resellerUserID)
	if availAfterRelease.Amount != 300_000 {
		t.Errorf("expected available balance 300,000 after release, got %d", availAfterRelease.Amount)
	}
}

func TestConcurrentDoubleSpendPrevention(t *testing.T) {
	repo := newInMemoryWalletRepo()
	svc := wallet.NewService(repo)
	ctx := context.Background()

	instanceID := int64(1)
	userID := int64(10)
	tgID := int64(100)

	// User has 100,000 Toman
	_, err := svc.Credit(ctx, instanceID, userID, tgID, money.NewToman(100_000), domain.TxTypeTopup, "topup", "Initial balance")
	if err != nil {
		t.Fatalf("credit failed: %v", err)
	}

	// 10 concurrent requests to debit 20,000 each.
	// Exactly 5 should succeed (5 * 20k = 100k), and 5 must fail with ErrInsufficientFunds.
	var wg sync.WaitGroup
	successCount := 0
	var countMu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := svc.Debit(ctx, instanceID, userID, tgID, money.NewToman(20_000), domain.TxTypePurchase, fmt.Sprintf("order_%d", idx), "Concurrent purchase")
			if err == nil {
				countMu.Lock()
				successCount++
				countMu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if successCount != 5 {
		t.Errorf("expected exactly 5 successful debits, got %d", successCount)
	}

	finalBalance, _ := repo.GetBalance(ctx, instanceID, userID)
	if finalBalance.Amount != 0 {
		t.Errorf("expected final balance 0, got %d", finalBalance.Amount)
	}
}
