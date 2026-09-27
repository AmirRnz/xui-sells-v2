package domain

import (
	"time"

	"xui-sells-v2/internal/domain/money"
)

// TxType represents the classification of a wallet transaction.
type TxType string

const (
	TxTypeTopup              TxType = "topup"
	TxTypePurchase           TxType = "purchase"
	TxTypeRenewal            TxType = "renewal"
	TxTypeRefund             TxType = "refund"
	TxTypeReferralReward     TxType = "referral_reward"
	TxTypeReferralClawback   TxType = "referral_clawback"
	TxTypeManualCredit       TxType = "manual_credit"
	TxTypeWholesaleDeduction TxType = "wholesale_deduction"
)

// WalletTransaction records an immutable balance movement in a user's wallet.
type WalletTransaction struct {
	ID           int64       `json:"id"`
	InstanceID   int64       `json:"instance_id"`
	UserID       int64       `json:"user_id"`
	TelegramID   int64       `json:"telegram_id"`
	Type         TxType      `json:"type"`
	Amount       money.Money `json:"amount"` // Positive for credit, negative for debit
	BalanceAfter money.Money `json:"balance_after"`
	ReferenceID  string      `json:"reference_id,omitempty"` // e.g. order_id or refund_id
	Description  string      `json:"description,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

// ReservationStatus defines the status of a shared reseller credit reservation.
type ReservationStatus string

const (
	ReservationStatusPending   ReservationStatus = "pending"
	ReservationStatusSettled   ReservationStatus = "settled"
	ReservationStatusCancelled ReservationStatus = "cancelled"
)

// ResellerFundingReservation represents an atomic hold on a reseller's wallet balance
// to fund a purchase on one of their child customer bots (P20).
type ResellerFundingReservation struct {
	ID                 int64             `json:"id"`
	ParentInstanceID   int64             `json:"parent_instance_id"`
	ChildInstanceID    int64             `json:"child_instance_id"`
	ResellerTelegramID int64             `json:"reseller_telegram_id"`
	OrderID            int64             `json:"order_id"`
	Amount             money.Money       `json:"amount"`
	Status             ReservationStatus `json:"status"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}
