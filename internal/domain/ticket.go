package domain

import "time"

// TicketStatus defines the current status of a support ticket.
type TicketStatus string

const (
	TicketStatusOpen     TicketStatus = "open"
	TicketStatusAnswered TicketStatus = "answered"
	TicketStatusClosed   TicketStatus = "closed"
)

// Ticket represents a support thread between a user and bot administrators.
type Ticket struct {
	ID         int64        `json:"id"`
	InstanceID int64        `json:"instance_id"`
	UserID     int64        `json:"user_id"`
	TelegramID int64        `json:"telegram_id"`
	Subject    string       `json:"subject"`
	Status     TicketStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// TicketMessage represents a message within a support ticket.
type TicketMessage struct {
	ID               int64     `json:"id"`
	TicketID         int64     `json:"ticket_id"`
	SenderTelegramID int64     `json:"sender_telegram_id"`
	IsAdmin          bool      `json:"is_admin"`
	Message          string    `json:"message"`
	AttachmentPath   string    `json:"attachment_path,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
