package broadcast_test

import (
	"context"
	"errors"
	"testing"

	"xui-sells-v2/internal/app/broadcast"
	"xui-sells-v2/internal/domain"
)

type mockBroadcastRepo struct {
	allUsers    []broadcast.Recipient
	subscribers []broadcast.Recipient
}

func (m *mockBroadcastRepo) GetRecipients(ctx context.Context, instanceID int64, audience broadcast.AudienceType) ([]broadcast.Recipient, error) {
	if audience == broadcast.AudienceActiveSubscribers {
		return m.subscribers, nil
	}
	return m.allUsers, nil
}

type mockSender struct {
	sent   []int64
	failOn int64
}

func (m *mockSender) SendMessage(ctx context.Context, instanceID, telegramID int64, text string) error {
	if telegramID == m.failOn {
		return errors.New("blocked by user")
	}
	m.sent = append(m.sent, telegramID)
	return nil
}

func TestBroadcastAudienceFiltering(t *testing.T) {
	repo := &mockBroadcastRepo{
		allUsers: []broadcast.Recipient{
			{TelegramID: 101, Language: domain.LangFA},
			{TelegramID: 102, Language: domain.LangEN},
			{TelegramID: 103, Language: domain.LangFA},
		},
		subscribers: []broadcast.Recipient{
			{TelegramID: 101, Language: domain.LangFA},
		},
	}

	sender := &mockSender{failOn: 102}
	svc := broadcast.NewService(repo, sender)
	ctx := context.Background()

	// 1. Broadcast to Active Subscribers only
	resSub, err := svc.DispatchBroadcast(ctx, 1, broadcast.AudienceActiveSubscribers, "Service upgrade notice")
	if err != nil {
		t.Fatalf("dispatch subscribers failed: %v", err)
	}
	if resSub.TotalRecipients != 1 || resSub.SentCount != 1 || resSub.FailedCount != 0 {
		t.Errorf("unexpected subscribers result: %+v", resSub)
	}

	// 2. Broadcast to All Users (with 1 failure on 102)
	sender.sent = nil
	resAll, err := svc.DispatchBroadcast(ctx, 1, broadcast.AudienceAllUsers, "Global announcement")
	if err != nil {
		t.Fatalf("dispatch all failed: %v", err)
	}
	if resAll.TotalRecipients != 3 || resAll.SentCount != 2 || resAll.FailedCount != 1 {
		t.Errorf("unexpected all users result: %+v", resAll)
	}
}
