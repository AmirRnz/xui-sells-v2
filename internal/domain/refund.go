package domain

import (
	"time"

	"xui-sells-v2/internal/domain/money"
)

// RefundStatus defines the state of a customer refund request.
type RefundStatus string

const (
	RefundStatusPending  RefundStatus = "pending"
	RefundStatusApproved RefundStatus = "approved"
	RefundStatusRejected RefundStatus = "rejected"
)

// RefundRequest represents a request submitted by a customer for a service refund.
type RefundRequest struct {
	ID              int64        `json:"id"`
	InstanceID      int64        `json:"instance_id"`
	UserID          int64        `json:"user_id"`
	TelegramID      int64        `json:"telegram_id"`
	ServiceID       int64        `json:"service_id"`
	OrderID         int64        `json:"order_id"`
	Amount          money.Money  `json:"amount"`
	Reason          string       `json:"reason"`
	AttachmentPath  string       `json:"attachment_path,omitempty"`
	Status          RefundStatus `json:"status"`
	RejectionReason string       `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}
