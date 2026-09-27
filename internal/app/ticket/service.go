package ticket

import (
	"context"
	"errors"
	"fmt"
	"time"

	"xui-sells-v2/internal/domain"
)

var (
	ErrTicketNotFound = errors.New("ticket not found")
	ErrEmptyMessage   = errors.New("ticket message cannot be empty")
	ErrTicketClosed   = errors.New("cannot add message to a closed ticket")
)

// TicketRepository abstracts ticket storage.
type TicketRepository interface {
	CreateTicket(ctx context.Context, t domain.Ticket) (*domain.Ticket, error)
	GetTicket(ctx context.Context, ticketID int64) (*domain.Ticket, error)
	ListTickets(ctx context.Context, instanceID int64, userID *int64, status *domain.TicketStatus) ([]domain.Ticket, error)
	UpdateTicketStatus(ctx context.Context, ticketID int64, status domain.TicketStatus) error
	CreateMessage(ctx context.Context, msg domain.TicketMessage) (*domain.TicketMessage, error)
	ListMessages(ctx context.Context, ticketID int64) ([]domain.TicketMessage, error)
}

// Service manages the lifecycle of customer support tickets.
type Service struct {
	repo TicketRepository
}

// NewService creates a new ticket service.
func NewService(repo TicketRepository) *Service {
	return &Service{repo: repo}
}

// CreateTicket opens a new ticket thread and persists the first message.
func (s *Service) CreateTicket(ctx context.Context, instanceID, userID, tgID int64, subject, initialMessage string) (*domain.Ticket, *domain.TicketMessage, error) {
	if initialMessage == "" {
		return nil, nil, ErrEmptyMessage
	}

	t := domain.Ticket{
		InstanceID: instanceID,
		UserID:     userID,
		TelegramID: tgID,
		Subject:    subject,
		Status:     domain.TicketStatusOpen,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	createdTicket, err := s.repo.CreateTicket(ctx, t)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	msg := domain.TicketMessage{
		TicketID:         createdTicket.ID,
		SenderTelegramID: tgID,
		IsAdmin:          false,
		Message:          initialMessage,
		CreatedAt:        time.Now(),
	}

	createdMsg, err := s.repo.CreateMessage(ctx, msg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create initial ticket message: %w", err)
	}

	return createdTicket, createdMsg, nil
}

// ListTickets returns tickets filtered by user or status.
func (s *Service) ListTickets(ctx context.Context, instanceID int64, userID *int64, status *domain.TicketStatus) ([]domain.Ticket, error) {
	return s.repo.ListTickets(ctx, instanceID, userID, status)
}

// GetTicket retrieves a ticket and all its messages.
func (s *Service) GetTicket(ctx context.Context, ticketID int64) (*domain.Ticket, []domain.TicketMessage, error) {
	t, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	if t == nil {
		return nil, nil, ErrTicketNotFound
	}

	msgs, err := s.repo.ListMessages(ctx, ticketID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list ticket messages: %w", err)
	}

	return t, msgs, nil
}

// AddMessage appends a message from the customer.
func (s *Service) AddMessage(ctx context.Context, ticketID, senderTgID int64, message, attachmentPath string) (*domain.TicketMessage, error) {
	if message == "" {
		return nil, ErrEmptyMessage
	}

	t, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}
	if t.Status == domain.TicketStatusClosed {
		return nil, ErrTicketClosed
	}

	msg := domain.TicketMessage{
		TicketID:         ticketID,
		SenderTelegramID: senderTgID,
		IsAdmin:          false,
		Message:          message,
		AttachmentPath:   attachmentPath,
		CreatedAt:        time.Now(),
	}

	createdMsg, err := s.repo.CreateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	_ = s.repo.UpdateTicketStatus(ctx, ticketID, domain.TicketStatusOpen)
	return createdMsg, nil
}

// ReplyTicket records an administrator response and transitions ticket to answered.
func (s *Service) ReplyTicket(ctx context.Context, ticketID, adminTgID int64, message string) (*domain.TicketMessage, error) {
	if message == "" {
		return nil, ErrEmptyMessage
	}

	t, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTicketNotFound
	}
	if t.Status == domain.TicketStatusClosed {
		return nil, ErrTicketClosed
	}

	msg := domain.TicketMessage{
		TicketID:         ticketID,
		SenderTelegramID: adminTgID,
		IsAdmin:          true,
		Message:          message,
		CreatedAt:        time.Now(),
	}

	createdMsg, err := s.repo.CreateMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateTicketStatus(ctx, ticketID, domain.TicketStatusAnswered); err != nil {
		return nil, fmt.Errorf("failed to update ticket status: %w", err)
	}

	return createdMsg, nil
}

// CloseTicket resolves and closes a support ticket.
func (s *Service) CloseTicket(ctx context.Context, ticketID int64) error {
	t, err := s.repo.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	if t == nil {
		return ErrTicketNotFound
	}
	return s.repo.UpdateTicketStatus(ctx, ticketID, domain.TicketStatusClosed)
}
