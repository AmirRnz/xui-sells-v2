package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	adapterHTTP "xui-sells-v2/internal/adapter/http"
	"xui-sells-v2/internal/app/stats"
	"xui-sells-v2/internal/app/ticket"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

type mockResellerStore struct {
	accounts map[int64]*adapterHTTP.ResellerAccount
}

func (m *mockResellerStore) GetAccountByTgID(ctx context.Context, tgID int64) (*adapterHTTP.ResellerAccount, error) {
	return m.accounts[tgID], nil
}

func (m *mockResellerStore) UpdatePasswordHash(ctx context.Context, tgID int64, newHash string) error {
	if a, ok := m.accounts[tgID]; ok {
		a.PasswordHash = newHash
	}
	return nil
}

type mockServiceStore struct {
	services []domain.Service
}

func (m *mockServiceStore) ListResellerServices(ctx context.Context, instanceID int64, search string, status string) ([]domain.Service, error) {
	return m.services, nil
}

func (m *mockServiceStore) GetServiceByID(ctx context.Context, id int64) (*domain.Service, error) {
	for _, s := range m.services {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockServiceStore) GetServiceByEmail(ctx context.Context, email string) (*domain.Service, error) {
	for _, s := range m.services {
		if s.ClientEmail == email {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockServiceStore) ResetTraffic(ctx context.Context, idOrEmail string) error {
	return nil
}

func (m *mockServiceStore) RotateSubID(ctx context.Context, serviceID int64) (*domain.Service, string, error) {
	for i, s := range m.services {
		if s.ID == serviceID {
			m.services[i].SubID = "rotated_new_sub"
			return &m.services[i], "https://sub.turbovpn.top/sub/rotated_new_sub", nil
		}
	}
	return nil, "", nil
}

type mockOrderStore struct {
	orders []domain.Order
}

func (m *mockOrderStore) ListOrders(ctx context.Context, instanceID int64, status string) ([]domain.Order, error) {
	return m.orders, nil
}

func (m *mockOrderStore) ApproveOrder(ctx context.Context, orderID int64) error {
	return nil
}

func (m *mockOrderStore) RejectOrder(ctx context.Context, orderID int64, reason string) error {
	return nil
}

type mockChildBotStore struct {
	bots []domain.Instance
}

func (m *mockChildBotStore) ListChildBots(ctx context.Context, parentInstanceID int64) ([]domain.Instance, error) {
	return m.bots, nil
}

type mockStatsRepo struct{}

func (m *mockStatsRepo) GetOperationalStats(ctx context.Context, instanceID int64) (*stats.OperationalStats, error) {
	return &stats.OperationalStats{
		RegisteredUsers: 100,
		ActiveServices:  80,
		TotalRevenue:    money.NewToman(5_000_000),
	}, nil
}

type mockTicketRepo struct {
	tickets []domain.Ticket
}

func (m *mockTicketRepo) CreateTicket(ctx context.Context, t domain.Ticket) (*domain.Ticket, error) {
	return &t, nil
}

func (m *mockTicketRepo) GetTicket(ctx context.Context, ticketID int64) (*domain.Ticket, error) {
	for _, t := range m.tickets {
		if t.ID == ticketID {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTicketRepo) ListTickets(ctx context.Context, instanceID int64, userID *int64, status *domain.TicketStatus) ([]domain.Ticket, error) {
	return m.tickets, nil
}

func (m *mockTicketRepo) UpdateTicketStatus(ctx context.Context, ticketID int64, status domain.TicketStatus) error {
	return nil
}

func (m *mockTicketRepo) CreateMessage(ctx context.Context, msg domain.TicketMessage) (*domain.TicketMessage, error) {
	return &msg, nil
}

func (m *mockTicketRepo) ListMessages(ctx context.Context, ticketID int64) ([]domain.TicketMessage, error) {
	return nil, nil
}

func setupTestServer(t *testing.T) (*adapterHTTP.Server, *mockResellerStore) {
	t.Helper()

	hash, err := adapterHTTP.HashPassword("secret123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	resellerStore := &mockResellerStore{
		accounts: map[int64]*adapterHTTP.ResellerAccount{
			999: {
				TelegramID:   999,
				InstanceID:   1,
				UserID:       10,
				Username:     "ultimate_reseller",
				FirstName:    "Alex",
				PasswordHash: hash,
				Tier:         domain.ResellerTierUltimate,
				ServiceName:  "TurboVPN Pro",
			},
			888: {
				TelegramID:   888,
				InstanceID:   2,
				UserID:       20,
				Username:     "pro_reseller",
				FirstName:    "Bob",
				PasswordHash: hash,
				Tier:         domain.ResellerTierPro, // Not entitled to Web Panel per D07
				ServiceName:  "BasicVPN",
			},
		},
	}

	serviceStore := &mockServiceStore{
		services: []domain.Service{
			{
				ID:          1,
				ClientEmail: "srv1@xui.net",
				SubID:       "sub1",
				LimitIP:     2,
			},
		},
	}

	orderStore := &mockOrderStore{
		orders: []domain.Order{
			{
				ID:     1,
				Amount: money.NewToman(100_000),
				Status: domain.OrderStatusPendingApproval,
			},
		},
	}

	childBotStore := &mockChildBotStore{
		bots: []domain.Instance{
			{
				ID:   10,
				Name: "ChildBot1",
				Type: domain.InstanceTypeChildCustomer,
			},
		},
	}

	mockFS := fstest.MapFS{
		"index.html": {
			Data: []byte("<!DOCTYPE html><html><body><h1>Reseller Panel</h1></body></html>"),
		},
		"assets/app.js": {
			Data: []byte("console.log('web panel');"),
		},
	}

	statsSvc := stats.NewService(&mockStatsRepo{})
	ticketSvc := ticket.NewService(&mockTicketRepo{
		tickets: []domain.Ticket{
			{ID: 1, Subject: "Help", Status: domain.TicketStatusOpen},
		},
	})

	server := adapterHTTP.NewServer(adapterHTTP.Config{
		ResellerStore: resellerStore,
		ServiceStore:  serviceStore,
		OrderStore:    orderStore,
		ChildBotStore: childBotStore,
		StatsSvc:      statsSvc,
		TicketSvc:     ticketSvc,
		WebFS:         mockFS,
	})

	return server, resellerStore
}

func TestAuthAndLoginFlow(t *testing.T) {
	server, _ := setupTestServer(t)

	// 1. Invalid Password -> 401
	loginBad := adapterHTTP.LoginRequest{TelegramID: 999, Password: "wrong"}
	body, _ := json.Marshal(loginBad)
	req := httptest.NewRequest(http.MethodPost, "/api/reseller/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rec.Code)
	}

	// 2. Valid Login -> 200 with token and cookie
	loginGood := adapterHTTP.LoginRequest{TelegramID: 999, Password: "secret123"}
	body, _ = json.Marshal(loginGood)
	req = httptest.NewRequest(http.MethodPost, "/api/reseller/login", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", rec.Code)
	}

	var resp adapterHTTP.LoginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	token := resp.Token
	if token == "" || resp.User.TelegramID != 999 {
		t.Errorf("unexpected login response: %+v", resp)
	}

	// 3. /me with token
	req = httptest.NewRequest(http.MethodGet, "/api/reseller/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /me, got %d", rec.Code)
	}

	// 4. Change Password
	changeReq := adapterHTTP.ChangePasswordRequest{
		OldPassword: "secret123",
		NewPassword: "newsecret456",
	}
	body, _ = json.Marshal(changeReq)
	req = httptest.NewRequest(http.MethodPost, "/api/reseller/change-password", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on change-password, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestWebPanelTierGating(t *testing.T) {
	server, _ := setupTestServer(t)

	// Login with Bob (Pro tier, tier 1)
	loginPro := adapterHTTP.LoginRequest{TelegramID: 888, Password: "secret123"}
	body, _ := json.Marshal(loginPro)
	req := httptest.NewRequest(http.MethodPost, "/api/reseller/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 login for pro user, got %d", rec.Code)
	}

	var respPro adapterHTTP.LoginResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &respPro)

	// Accessing /api/reseller/stats with Pro tier should be 403 Forbidden (requires Ultimate tier per D07)
	req = httptest.NewRequest(http.MethodGet, "/api/reseller/stats", nil)
	req.Header.Set("Authorization", "Bearer "+respPro.Token)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-Ultimate reseller accessing web panel stats, got %d", rec.Code)
	}

	// Login with Alex (Ultimate tier, tier 2)
	loginUltimate := adapterHTTP.LoginRequest{TelegramID: 999, Password: "secret123"}
	body, _ = json.Marshal(loginUltimate)
	req = httptest.NewRequest(http.MethodPost, "/api/reseller/login", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	var respUltimate adapterHTTP.LoginResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &respUltimate)

	// Accessing /api/reseller/stats with Ultimate tier -> 200 OK!
	req = httptest.NewRequest(http.MethodGet, "/api/reseller/stats", nil)
	req.Header.Set("Authorization", "Bearer "+respUltimate.Token)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for Ultimate reseller accessing stats, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPanelOperations(t *testing.T) {
	server, _ := setupTestServer(t)

	// Login Alex
	login := adapterHTTP.LoginRequest{TelegramID: 999, Password: "secret123"}
	body, _ := json.Marshal(login)
	req := httptest.NewRequest(http.MethodPost, "/api/reseller/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	var resp adapterHTTP.LoginResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	token := resp.Token

	endpoints := []struct {
		method string
		url    string
		body   string
	}{
		{http.MethodGet, "/api/reseller/services", ""},
		{http.MethodPost, "/api/reseller/services/1/reset-traffic", ""},
		{http.MethodPost, "/api/reseller/services/1/rotate-sub", ""},
		{http.MethodGet, "/api/reseller/orders", ""},
		{http.MethodPost, "/api/reseller/orders/1/approve", ""},
		{http.MethodPost, "/api/reseller/orders/1/reject", `{"reason":"fraud"}`},
		{http.MethodGet, "/api/reseller/child-bots", ""},
		{http.MethodGet, "/api/reseller/tickets", ""},
		{http.MethodPost, "/api/reseller/tickets/1/reply", `{"message":"Hello"}`},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.url, func(t *testing.T) {
			var rBody *bytes.Reader
			if ep.body != "" {
				rBody = bytes.NewReader([]byte(ep.body))
			} else {
				rBody = bytes.NewReader(nil)
			}
			req := httptest.NewRequest(ep.method, ep.url, rBody)
			req.Header.Set("Authorization", "Bearer "+token)
			if ep.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			server.Router().ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d: %s", ep.url, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestStaticSPAServing(t *testing.T) {
	server, _ := setupTestServer(t)

	// 1. Root /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Reseller Panel")) {
		t.Errorf("failed to serve root index.html: %s", rec.Body.String())
	}

	// 2. Client-side route /dashboard -> SPA fallback serves index.html
	req = httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Reseller Panel")) {
		t.Errorf("failed to serve SPA fallback index.html: %s", rec.Body.String())
	}

	// 3. Static asset /assets/app.js
	req = httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("console.log")) {
		t.Errorf("failed to serve static asset app.js: %s", rec.Body.String())
	}
}
