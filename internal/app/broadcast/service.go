package broadcast

import (
	"context"
	"errors"
	"fmt"

	"xui-sells-v2/internal/domain"
)

var (
	ErrEmptyMessage = errors.New("broadcast message cannot be empty")
)

// AudienceType defines the recipient filter for broadcasts (P13).
type AudienceType string

const (
	AudienceAllUsers          AudienceType = "all"
	AudienceActiveSubscribers AudienceType = "active_subscribers"
)

// Recipient holds recipient Telegram ID and language preference.
type Recipient struct {
	TelegramID int64
	Language   domain.Language
}

// BroadcastRepository abstracts user query by subscription status.
type BroadcastRepository interface {
	GetRecipients(ctx context.Context, instanceID int64, audience AudienceType) ([]Recipient, error)
}

// MessageSender sends the actual message to Telegram.
type MessageSender interface {
	SendMessage(ctx context.Context, instanceID int64, telegramID int64, text string) error
}

// Result summarizes the broadcast outcome.
type Result struct {
	TotalRecipients int
	SentCount       int
	FailedCount     int
}

// Service manages mass notification dispatch.
type Service struct {
	repo   BroadcastRepository
	sender MessageSender
}

// NewService creates a new broadcast service.
func NewService(repo BroadcastRepository, sender MessageSender) *Service {
	return &Service{
		repo:   repo,
		sender: sender,
	}
}

// DispatchBroadcast broadcasts a message to either all users or only active subscribers (P13).
func (s *Service) DispatchBroadcast(ctx context.Context, instanceID int64, audience AudienceType, message string) (*Result, error) {
	if message == "" {
		return nil, ErrEmptyMessage
	}

	recipients, err := s.repo.GetRecipients(ctx, instanceID, audience)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipients for audience %s: %w", audience, err)
	}

	res := &Result{
		TotalRecipients: len(recipients),
	}

	for _, r := range recipients {
		if err := s.sender.SendMessage(ctx, instanceID, r.TelegramID, message); err != nil {
			res.FailedCount++
		} else {
			res.SentCount++
		}
	}

	return res, nil
}
