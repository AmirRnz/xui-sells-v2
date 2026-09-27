package domain

import (
	"time"

	"xui-sells-v2/internal/domain/money"
)

// OrderType specifies the kind of operation represented by an order.
type OrderType string

const (
	OrderTypePurchase        OrderType = "purchase"
	OrderTypeRenewal         OrderType = "renewal"
	OrderTypeUserCountChange OrderType = "user_count_change"
	OrderTypeTopup           OrderType = "topup"
)

// PaymentMethod specifies how the order was paid or is pending payment.
type PaymentMethod string

const (
	PaymentMethodWallet PaymentMethod = "wallet"
	PaymentMethodDirect PaymentMethod = "direct"
)

// OrderStatus defines the lifecycle status of an order.
type OrderStatus string

const (
	OrderStatusPendingApproval OrderStatus = "pending_approval"
	OrderStatusApproved        OrderStatus = "approved"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusCancelled       OrderStatus = "cancelled"
)

// Order represents a financial or subscription transaction order.
type Order struct {
	ID              int64         `json:"id"`
	InstanceID      int64         `json:"instance_id"`
	UserID          int64         `json:"user_id"`
	TelegramID      int64         `json:"telegram_id"`
	ServiceID       *int64        `json:"service_id,omitempty"`
	PlanID          *int64        `json:"plan_id,omitempty"`
	Type            OrderType     `json:"type"`
	DurationMonths  int           `json:"duration_months"`
	UserCount       int           `json:"user_count"`
	Amount          money.Money   `json:"amount"`
	PaymentMethod   PaymentMethod `json:"payment_method"`
	Status          OrderStatus   `json:"status"`
	ReceiptImageURL string        `json:"receipt_image_url,omitempty"`
	ReceiptNotes    string        `json:"receipt_notes,omitempty"`
	RejectionReason string        `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}
