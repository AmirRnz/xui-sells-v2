package wallet

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

var (
	ErrInsufficientFunds         = errors.New("insufficient wallet balance")
	ErrInsufficientResellerFunds = errors.New("insufficient provider credit; please contact support")
	ErrInvalidReservationState   = errors.New("reservation is not in a pending state")
	ErrReservationNotFound       = errors.New("reservation not found")
	ErrNegativeAmount            = errors.New("amount must be strictly positive")
)

// WalletRepository abstracts persistence and transactional locking for balances and reservations.
type WalletRepository interface {
	GetBalance(ctx context.Context, instanceID int64, userID int64) (money.Money, error)
	GetAvailableBalance(ctx context.Context, instanceID int64, userID int64) (money.Money, error)
	RecordTransaction(ctx context.Context, tx domain.WalletTransaction) error
	CreateReservation(ctx context.Context, res domain.ResellerFundingReservation) (*domain.ResellerFundingReservation, error)
	GetReservation(ctx context.Context, reservationID int64) (*domain.ResellerFundingReservation, error)
	UpdateReservationStatus(ctx context.Context, reservationID int64, status domain.ReservationStatus) error
}

// Service provides wallet ledger operations and shared reseller credit management.
type Service struct {
	repo WalletRepository
	mu   sync.Mutex
}

// NewService creates a new wallet service.
func NewService(repo WalletRepository) *Service {
	return &Service{repo: repo}
}

// Credit adds funds to a user's wallet and records an immutable ledger posting.
func (s *Service) Credit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error) {
	if !amount.IsPositive() {
		return nil, ErrNegativeAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	balance, err := s.repo.GetBalance(ctx, instanceID, userID)
	if err != nil {
		balance = money.New(0, amount.Currency)
	}

	newBalance, err := balance.Add(amount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate new balance: %w", err)
	}

	tx := domain.WalletTransaction{
		InstanceID:   instanceID,
		UserID:       userID,
		TelegramID:   tgID,
		Type:         txType,
		Amount:       amount,
		BalanceAfter: newBalance,
		ReferenceID:  refID,
		Description:  desc,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.RecordTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to record credit transaction: %w", err)
	}

	return &tx, nil
}

// Debit deducts funds from a user's wallet with an atomic balance check.
func (s *Service) Debit(ctx context.Context, instanceID int64, userID int64, tgID int64, amount money.Money, txType domain.TxType, refID string, desc string) (*domain.WalletTransaction, error) {
	if !amount.IsPositive() {
		return nil, ErrNegativeAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	balance, err := s.repo.GetBalance(ctx, instanceID, userID)
	if err != nil {
		return nil, ErrInsufficientFunds
	}

	if balance.LessThan(amount) {
		return nil, ErrInsufficientFunds
	}

	newBalance, err := balance.Sub(amount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate new balance: %w", err)
	}

	debitAmount := money.New(-amount.Amount, amount.Currency)
	tx := domain.WalletTransaction{
		InstanceID:   instanceID,
		UserID:       userID,
		TelegramID:   tgID,
		Type:         txType,
		Amount:       debitAmount,
		BalanceAfter: newBalance,
		ReferenceID:  refID,
		Description:  desc,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.RecordTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to record debit transaction: %w", err)
	}

	return &tx, nil
}

// ReserveResellerCredit places an atomic hold on a reseller's wallet balance
// to guarantee wholesale payment for a child customer order (P20 & D06).
func (s *Service) ReserveResellerCredit(ctx context.Context, parentInstanceID int64, childInstanceID int64, resellerTgID int64, resellerUserID int64, orderID int64, amount money.Money) (*domain.ResellerFundingReservation, error) {
	if !amount.IsPositive() {
		return nil, ErrNegativeAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	available, err := s.repo.GetAvailableBalance(ctx, parentInstanceID, resellerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reseller available balance: %w", err)
	}

	if available.LessThan(amount) {
		return nil, ErrInsufficientResellerFunds
	}

	reservation := domain.ResellerFundingReservation{
		ParentInstanceID:   parentInstanceID,
		ChildInstanceID:    childInstanceID,
		ResellerTelegramID: resellerTgID,
		OrderID:            orderID,
		Amount:             amount,
		Status:             domain.ReservationStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	created, err := s.repo.CreateReservation(ctx, reservation)
	if err != nil {
		return nil, fmt.Errorf("failed to create reseller credit reservation: %w", err)
	}

	return created, nil
}

// SettleReservation finalizes the deduction of the wholesale cost from the reseller's wallet.
func (s *Service) SettleReservation(ctx context.Context, reservationID int64, resellerUserID int64) error {
	res, err := s.repo.GetReservation(ctx, reservationID)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrReservationNotFound
	}

	if res.Status == domain.ReservationStatusSettled {
		return nil // Idempotent
	}
	if res.Status != domain.ReservationStatusPending {
		return fmt.Errorf("%w: status is %s", ErrInvalidReservationState, res.Status)
	}

	refID := fmt.Sprintf("order_%d", res.OrderID)
	desc := fmt.Sprintf("Wholesale deduction for child bot order #%d", res.OrderID)

	_, err = s.Debit(ctx, res.ParentInstanceID, resellerUserID, res.ResellerTelegramID, res.Amount, domain.TxTypeWholesaleDeduction, refID, desc)
	if err != nil {
		return fmt.Errorf("failed to debit reseller wallet on settlement: %w", err)
	}

	if err := s.repo.UpdateReservationStatus(ctx, reservationID, domain.ReservationStatusSettled); err != nil {
		return fmt.Errorf("failed to update reservation status: %w", err)
	}

	return nil
}

// ReleaseReservation cancels the hold on reseller funds (e.g. if the child bot order was rejected).
func (s *Service) ReleaseReservation(ctx context.Context, reservationID int64) error {
	res, err := s.repo.GetReservation(ctx, reservationID)
	if err != nil {
		return err
	}
	if res == nil {
		return ErrReservationNotFound
	}

	if res.Status == domain.ReservationStatusCancelled {
		return nil // Idempotent
	}
	if res.Status != domain.ReservationStatusPending {
		return fmt.Errorf("%w: status is %s", ErrInvalidReservationState, res.Status)
	}

	return s.repo.UpdateReservationStatus(ctx, reservationID, domain.ReservationStatusCancelled)
}
