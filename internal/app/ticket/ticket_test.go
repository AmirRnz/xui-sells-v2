package ticket_test

import (
	"context"
	"testing"
	"time"

	"xui-sells-v2/internal/app/ticket"
	"xui-sells-v2/internal/domain"
)

type mockTicketRepo struct {
	tickets  map[int64]*domain.Ticket
	messages map[int64][]domain.TicketMessage
	nextTID  int64
	nextMID  int64
}

func newMockTicketRepo() *mockTicketRepo {
	return &mockTicketRepo{
		tickets:  make(map[int64]*domain.Ticket),
		messages: make(map[int64][]domain.TicketMessage),
		nextTID:  1,
		nextMID:  1,
	}
}

func (m *mockTicketRepo) CreateTicket(ctx context.Context, t domain.Ticket) (*domain.Ticket, error) {
	t.ID = m.nextTID
	m.nextTID++
	cp := t
	m.tickets[t.ID] = &cp
	return &cp, nil
}

func (m *mockTicketRepo) GetTicket(ctx context.Context, id int64) (*domain.Ticket, error) {
	t, ok := m.tickets[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *mockTicketRepo) ListTickets(ctx context.Context, instanceID int64, userID *int64, status *domain.TicketStatus) ([]domain.Ticket, error) {
	var res []domain.Ticket
	for _, t := range m.tickets {
		if t.InstanceID == instanceID {
			if userID != nil && t.UserID != *userID {
				continue
			}
			if status != nil && t.Status != *status {
				continue
			}
			res = append(res, *t)
		}
	}
	return res, nil
}

func (m *mockTicketRepo) UpdateTicketStatus(ctx context.Context, id int64, status domain.TicketStatus) error {
	t, ok := m.tickets[id]
	if !ok {
		return ticket.ErrTicketNotFound
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return nil
}

func (m *mockTicketRepo) CreateMessage(ctx context.Context, msg domain.TicketMessage) (*domain.TicketMessage, error) {
	msg.ID = m.nextMID
	m.nextMID++
	m.messages[msg.TicketID] = append(m.messages[msg.TicketID], msg)
	return &msg, nil
}

func (m *mockTicketRepo) ListMessages(ctx context.Context, ticketID int64) ([]domain.TicketMessage, error) {
	return m.messages[ticketID], nil
}

func TestTicketLifecycle(t *testing.T) {
	repo := newMockTicketRepo()
	svc := ticket.NewService(repo)
	ctx := context.Background()

	// 1. Create Ticket
	tkt, msg, err := svc.CreateTicket(ctx, 1, 10, 100, "Connection Problem", "Cannot connect to server")
	if err != nil {
		t.Fatalf("CreateTicket failed: %v", err)
	}
	if tkt.Status != domain.TicketStatusOpen || msg.Message != "Cannot connect to server" {
		t.Errorf("unexpected ticket state: %+v, %+v", tkt, msg)
	}

	// 2. Reply Ticket (Admin) -> transitions to Answered
	adminMsg, err := svc.ReplyTicket(ctx, tkt.ID, 999, "Please update your client app.")
	if err != nil {
		t.Fatalf("ReplyTicket failed: %v", err)
	}
	if !adminMsg.IsAdmin {
		t.Errorf("expected admin message flag")
	}

	// Check updated status
	fetched, msgs, err := svc.GetTicket(ctx, tkt.ID)
	if err != nil || fetched.Status != domain.TicketStatusAnswered {
		t.Errorf("expected answered status, got %s (err: %v)", fetched.Status, err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}

	// 3. User responds -> transitions back to Open
	_, err = svc.AddMessage(ctx, tkt.ID, 100, "Worked! Thanks.", "")
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}
	fetched2, _, _ := svc.GetTicket(ctx, tkt.ID)
	if fetched2.Status != domain.TicketStatusOpen {
		t.Errorf("expected ticket to re-open on user reply, got %s", fetched2.Status)
	}

	// 4. Close Ticket
	if err := svc.CloseTicket(ctx, tkt.ID); err != nil {
		t.Fatalf("CloseTicket failed: %v", err)
	}
	fetched3, _, _ := svc.GetTicket(ctx, tkt.ID)
	if fetched3.Status != domain.TicketStatusClosed {
		t.Errorf("expected closed status, got %s", fetched3.Status)
	}

	// Message to closed ticket should fail
	if _, err := svc.AddMessage(ctx, tkt.ID, 100, "Still there?", ""); err != ticket.ErrTicketClosed {
		t.Errorf("expected ErrTicketClosed, got %v", err)
	}
}
