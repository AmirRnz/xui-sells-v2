package telegram_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"xui-sells-v2/internal/adapter/telegram"
	"xui-sells-v2/internal/app/broadcast"
	"xui-sells-v2/internal/app/entitlement"
	"xui-sells-v2/internal/app/order"
	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/app/stats"
	"xui-sells-v2/internal/app/trial"
	"xui-sells-v2/internal/app/wallet"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
	"xui-sells-v2/internal/infra/xui"
)

type mockSender struct {
	sentMessages []string
	sentPhotos   []string
	lastMarkup   any
}

func (m *mockSender) SendMessage(ctx context.Context, chatID int64, text string, replyMarkup any) error {
	m.sentMessages = append(m.sentMessages, text)
	m.lastMarkup = replyMarkup
	return nil
}

func (m *mockSender) SendPhoto(ctx context.Context, chatID int64, photo []byte, caption string, replyMarkup any) error {
	m.sentPhotos = append(m.sentPhotos, caption)
	m.lastMarkup = replyMarkup
	return nil
}

type mockXUIClient struct{}

func (m *mockXUIClient) AddClient(ctx context.Context, req xui.AddClientRequest) error { return nil }
func (m *mockXUIClient) UpdateClient(ctx context.Context, email string, client xui.ClientPayload) error {
	return nil
}
func (m *mockXUIClient) DeleteClient(ctx context.Context, email string) error { return nil }
func (m *mockXUIClient) GetClient(ctx context.Context, email string) (*xui.ClientResponse, error) {
	return nil, nil
}
func (m *mockXUIClient) ListInbounds(ctx context.Context) ([]xui.Inbound, error) { return nil, nil }
func (m *mockXUIClient) GetSubLinks(ctx context.Context, subId string) ([]string, error) {
	return nil, nil
}
func (m *mockXUIClient) ResetTraffic(ctx context.Context, email string) error { return nil }
func (m *mockXUIClient) GetSettings(ctx context.Context) (*xui.AllSetting, error) {
	return &xui.AllSetting{SubURI: "https://sub.myvpn.com:8443/sub/"}, nil
}

type mockOrderRepo struct {
	orders []domain.Order
}

func (m *mockOrderRepo) CreateOrder(ctx context.Context, o domain.Order) (*domain.Order, error) {
	o.ID = int64(len(m.orders) + 1)
	m.orders = append(m.orders, o)
	return &o, nil
}
func (m *mockOrderRepo) GetOrder(ctx context.Context, id int64) (*domain.Order, error) { return nil, nil }
func (m *mockOrderRepo) UpdateOrderStatus(ctx context.Context, id int64, status domain.OrderStatus, reason string) error {
	return nil
}

type mockWalletRepo struct {
	balances     map[int64]money.Money
	reservations []domain.ResellerFundingReservation
}

func (m *mockWalletRepo) GetBalance(ctx context.Context, instanceID, userID int64) (money.Money, error) {
	if b, ok := m.balances[userID]; ok {
		return b, nil
	}
	return money.NewToman(0), nil
}

func (m *mockWalletRepo) GetAvailableBalance(ctx context.Context, instanceID, userID int64) (money.Money, error) {
	return m.GetBalance(ctx, instanceID, userID)
}

func (m *mockWalletRepo) RecordTransaction(ctx context.Context, tx domain.WalletTransaction) error {
	return nil
}

func (m *mockWalletRepo) CreateReservation(ctx context.Context, res domain.ResellerFundingReservation) (*domain.ResellerFundingReservation, error) {
	res.ID = int64(len(m.reservations) + 1)
	m.reservations = append(m.reservations, res)
	return &res, nil
}

func (m *mockWalletRepo) GetReservation(ctx context.Context, id int64) (*domain.ResellerFundingReservation, error) {
	return nil, nil
}

func (m *mockWalletRepo) UpdateReservationStatus(ctx context.Context, id int64, status domain.ReservationStatus) error {
	return nil
}

type mockTrialRepo struct{}

func (m *mockTrialRepo) GetLastTrialTime(ctx context.Context, instanceID string, userTgID, planID int64) (*time.Time, error) {
	return nil, nil
}
func (m *mockTrialRepo) CountTrialsInWindow(ctx context.Context, instanceID string, userTgID, planID int64, windowStart time.Time) (int, error) {
	return 0, nil
}
func (m *mockTrialRepo) CountDailyResellerTrials(ctx context.Context, instanceID string, since time.Time) (int, error) {
	return 0, nil
}
func (m *mockTrialRepo) RecordTrial(ctx context.Context, instanceID string, record domain.TrialRecord) error {
	return nil
}

type mockStatsRepo struct{}

func (m *mockStatsRepo) GetOperationalStats(ctx context.Context, instanceID int64) (*stats.OperationalStats, error) {
	return &stats.OperationalStats{
		RegisteredUsers: 50,
		ActiveServices:  30,
		TotalRevenue:    money.NewToman(2_000_000),
	}, nil
}

type mockBroadcastRepo struct{}

func (m *mockBroadcastRepo) GetRecipients(ctx context.Context, instanceID int64, audience broadcast.AudienceType) ([]broadcast.Recipient, error) {
	return []broadcast.Recipient{{TelegramID: 101, Language: domain.LangFA}}, nil
}

type mockBroadcastSender struct{}

func (m *mockBroadcastSender) SendMessage(ctx context.Context, instanceID, telegramID int64, text string) error {
	return nil
}

func setupTestBot(t *testing.T, inst domain.Instance) (*telegram.BotInstance, *mockSender, *mockWalletRepo) {
	sender := &mockSender{}
	walletRepo := &mockWalletRepo{
		balances: make(map[int64]money.Money),
	}

	walletSvc := wallet.NewService(walletRepo)
	provSvc := provisioning.NewService(&mockXUIClient{})
	trialSvc := trial.NewService(&mockTrialRepo{})
	orderSvc := order.NewService(&mockOrderRepo{}, nil, walletSvc, provSvc)
	entitlementSvc := entitlement.NewResolver()
	statsSvc := stats.NewService(&mockStatsRepo{})
	broadcastSvc := broadcast.NewService(&mockBroadcastRepo{}, &mockBroadcastSender{})
	supervisor := telegram.NewSupervisor()

	deps := telegram.BotDependencies{
		Sender:         sender,
		WalletSvc:      walletSvc,
		ProvSvc:        provSvc,
		TrialSvc:       trialSvc,
		OrderSvc:       orderSvc,
		EntitlementSvc: entitlementSvc,
		StatsSvc:       statsSvc,
		BroadcastSvc:   broadcastSvc,
		Supervisor:     supervisor,
	}

	b := telegram.NewBotInstance(inst, deps)

	// Setup plans
	b.SetPlans([]domain.Plan{
		{
			ID:             1,
			InstanceID:     inst.ID,
			Name:           "VIP Plan",
			BasePrice:      money.FromTomanInput(200),
			ExtraUserPrice: money.NewToman(50_000),
			BaseUsers:      1,
			MaxUsers:       5,
			Durations:      []int{1, 2, 3},
			TrafficBytes:   53687091200,
			TrialConfig: domain.TrialConfig{
				TrafficBytes:    1073741824,
				DurationSeconds: 86400,
			},
			IsActive: true,
		},
	})

	return b, sender, walletRepo
}

func TestSupervisorLifecycle(t *testing.T) {
	sup := telegram.NewSupervisor()

	inst1 := domain.Instance{ID: 1, Name: "Customer Bot", Type: domain.InstanceTypeCustomer}
	inst2 := domain.Instance{ID: 2, Name: "Reseller Bot", Type: domain.InstanceTypeReseller}
	parentID := int64(2)
	child1 := domain.Instance{ID: 3, Name: "Child Bot 1", Type: domain.InstanceTypeChildCustomer, ParentInstanceID: &parentID}

	b1 := telegram.NewBotInstance(inst1, telegram.BotDependencies{})
	b2 := telegram.NewBotInstance(inst2, telegram.BotDependencies{})
	bc := telegram.NewBotInstance(child1, telegram.BotDependencies{})

	_ = sup.RegisterBot(b1)
	_ = sup.RegisterBot(b2)
	_ = sup.RegisterBot(bc)

	if len(sup.ListRunning()) != 3 {
		t.Errorf("expected 3 running bots, got %d", len(sup.ListRunning()))
	}
	if sup.CountChildBots(2) != 1 {
		t.Errorf("expected 1 child bot under parent 2, got %d", sup.CountChildBots(2))
	}

	_ = sup.StopBot(1)
	if len(sup.ListRunning()) != 2 {
		t.Errorf("expected 2 running bots after stopping 1, got %d", len(sup.ListRunning()))
	}
}

func TestStartCommandReferralAndNoLanguageSelector(t *testing.T) {
	inst := domain.Instance{
		ID:          1,
		Name:        "FastVPN Bot",
		Type:        domain.InstanceTypeCustomer,
		DefaultLang: domain.LangFA,
	}
	bot, sender, _ := setupTestBot(t, inst)
	ctx := context.Background()

	// 1. /start with referral link parameter /start ref_777777 (P15)
	err := bot.HandleStart(ctx, 12345, "alice", "Alice", "", "/start ref_777777")
	if err != nil {
		t.Fatalf("HandleStart failed: %v", err)
	}

	user := bot.GetUser(12345)
	if user == nil || user.ReferrerTelegramID == nil || *user.ReferrerTelegramID != 777777 {
		t.Errorf("expected referrer ID 777777 recorded, got %+v", user)
	}

	// P25 INVARIANT: Never show language selector on /start!
	if sender.lastMarkup != nil {
		t.Errorf("P25 violation: language selector keyboard was shown on /start!")
	}

	// 2. /language command: The ONLY place where the language selector keyboard appears (P25)
	sender.lastMarkup = nil
	err = bot.HandleLanguage(ctx, 12345)
	if err != nil {
		t.Fatalf("HandleLanguage failed: %v", err)
	}
	if sender.lastMarkup == nil {
		t.Errorf("expected language keyboard presented on /language command")
	}

	// Change language to English
	err = bot.SetUserLanguage(ctx, 12345, domain.LangEN)
	if err != nil {
		t.Fatalf("SetUserLanguage failed: %v", err)
	}
	if bot.GetUser(12345).Language != domain.LangEN {
		t.Errorf("expected language updated to EN")
	}
}

func TestCustomerPurchaseWizard(t *testing.T) {
	inst := domain.Instance{
		ID:          1,
		Name:        "FastVPN Bot",
		Type:        domain.InstanceTypeCustomer,
		DefaultLang: domain.LangFA,
		CardNumber:  "6037-9911-2233-4455",
		CardHolder:  "Ali Rezaei",
	}
	bot, sender, _ := setupTestBot(t, inst)
	ctx := context.Background()
	tgID := int64(12345)
	_ = bot.HandleStart(ctx, tgID, "user1", "User", "", "/start")

	// Step 1: Select Plan
	err := bot.StartPurchaseWizard(ctx, tgID, 1)
	if err != nil {
		t.Fatalf("StartPurchaseWizard failed: %v", err)
	}

	// Step 2: Duration (1 month)
	_ = bot.ProcessPurchaseStep(ctx, tgID, "1")

	// Step 3: Users (2 users: base 1 + 1 extra = 200k + 50k = 250,000 Toman)
	_ = bot.ProcessPurchaseStep(ctx, tgID, "2")

	// Step 4: Name (MyVPN)
	_ = bot.ProcessPurchaseStep(ctx, tgID, "MyVPN")

	// Verify quote summary was presented
	lastMsg := sender.sentMessages[len(sender.sentMessages)-1]
	if !strings.Contains(lastMsg, "خلاصه سفارش") && !strings.Contains(lastMsg, "Order Summary") {
		t.Errorf("quote summary not shown: %s", lastMsg)
	}

	// Step 5: Choose Card payment (Direct)
	_ = bot.ProcessPurchaseStep(ctx, tgID, "card")
	cardMsg := sender.sentMessages[len(sender.sentMessages)-1]

	// Verify structured Persian bank card layout (P09)
	if !strings.Contains(cardMsg, "شماره کارت جهت واریز:") || !strings.Contains(cardMsg, "6037-9911-2233-4455") {
		t.Errorf("Persian card layout missing required structure: %s", cardMsg)
	}

	// Step 6: Submit receipt text
	_ = bot.ProcessPurchaseStep(ctx, tgID, "Paid ref: 9482104 Mellat")
	ack := sender.sentMessages[len(sender.sentMessages)-1]
	if !strings.Contains(ack, "رسید شما ثبت شد") && !strings.Contains(ack, "Receipt submitted") {
		t.Errorf("receipt confirmation missing: %s", ack)
	}
}

func TestChildBotSharedResellerCredit(t *testing.T) {
	parentID := int64(10)
	resellerAdminTgID := int64(999999)

	childInst := domain.Instance{
		ID:               2,
		Name:             "Reseller Child Bot",
		Type:             domain.InstanceTypeChildCustomer,
		ParentInstanceID: &parentID,
		AdminTelegramID:  resellerAdminTgID,
		DefaultLang:      domain.LangEN,
	}

	bot, sender, walletRepo := setupTestBot(t, childInst)
	ctx := context.Background()
	customerTgID := int64(55555)

	_ = bot.HandleStart(ctx, customerTgID, "cust", "Cust", "", "/start")
	_ = bot.StartPurchaseWizard(ctx, customerTgID, 1)
	_ = bot.ProcessPurchaseStep(ctx, customerTgID, "1")
	_ = bot.ProcessPurchaseStep(ctx, customerTgID, "1")
	_ = bot.ProcessPurchaseStep(ctx, customerTgID, "sub1")

	// Reseller has 0 balance in parent instance -> Checkout should fail with P20 error
	walletRepo.balances[customerTgID] = money.NewToman(500_000) // Customer has funds
	walletRepo.balances[resellerAdminTgID] = money.NewToman(0)  // Reseller has NO funds

	_ = bot.ProcessPurchaseStep(ctx, customerTgID, "wallet")

	// P20 Verification: Customer gets "Insufficient provider credit" error
	lastMsg := sender.sentMessages[len(sender.sentMessages)-2]
	if !strings.Contains(lastMsg, "Insufficient provider credit. Please contact support.") {
		t.Errorf("customer did not receive P20 error: %s", lastMsg)
	}

	// P20 Verification: Reseller gets notification alert in parent bot
	resellerNotice := sender.sentMessages[len(sender.sentMessages)-1]
	if !strings.Contains(resellerNotice, "Insufficient wallet balance to fund child bot") {
		t.Errorf("reseller alert not triggered: %s", resellerNotice)
	}
}

func TestFreeTrialAndUserCountDecreaseRotation(t *testing.T) {
	inst := domain.Instance{
		ID:          1,
		Name:        "FastVPN Bot",
		Type:        domain.InstanceTypeCustomer,
		DefaultLang: domain.LangEN,
		PanelURL:    "https://panel.example.com:2053",
	}
	bot, sender, _ := setupTestBot(t, inst)
	ctx := context.Background()
	tgID := int64(777)
	_ = bot.HandleStart(ctx, tgID, "trial_user", "Trial", "", "/start")

	plan, _ := bot.GetPlan(1)

	// 1. Issue Free Trial
	err := bot.RequestFreeTrial(ctx, tgID, plan, "")
	if err != nil {
		t.Fatalf("RequestFreeTrial failed: %v", err)
	}
	if len(sender.sentPhotos) != 1 {
		t.Fatalf("expected QR code delivered for trial")
	}

	// 2. Decrease User Count (P05 & Contract Section 5: Rotates subId and sends warning)
	activeService := domain.Service{
		ID:          10,
		ClientEmail: "user_sub@xui.net",
		SubID:       "old_sub_id",
		LimitIP:     3,
	}

	sender.sentPhotos = nil
	err = bot.HandleDecreaseUserCount(ctx, tgID, activeService, 2)
	if err != nil {
		t.Fatalf("HandleDecreaseUserCount failed: %v", err)
	}

	if len(sender.sentPhotos) != 1 {
		t.Fatalf("expected new QR code issued on user count decrease")
	}
	warning := sender.sentPhotos[0]
	if !strings.Contains(warning, "previous subscription link is now INVALID") {
		t.Errorf("P05 warning missing from rotated delivery: %s", warning)
	}
}

func TestResellerBotFirstActionMandatoryGroupName(t *testing.T) {
	inst := domain.Instance{
		ID:          5,
		Name:        "New Reseller Bot",
		Type:        domain.InstanceTypeReseller,
		GroupName:   "", // Unconfigured
		DefaultLang: domain.LangEN,
	}
	bot, sender, _ := setupTestBot(t, inst)
	ctx := context.Background()
	resellerTgID := int64(999999)

	// First /start
	_ = bot.HandleStart(ctx, resellerTgID, "reseller", "Reseller", "", "/start")
	prompt := sender.sentMessages[len(sender.sentMessages)-1]
	if !strings.Contains(prompt, "please enter your Service Name") {
		t.Errorf("reseller not prompted to choose service name on start: %s", prompt)
	}

	// Enter service name -> set group_name
	_ = bot.ProcessResellerStep(ctx, resellerTgID, "TurboVPN")
	if bot.Instance.GroupName != "TurboVPN" {
		t.Errorf("expected group name TurboVPN set, got %s", bot.Instance.GroupName)
	}

	// Web panel password setup (P21)
	_ = bot.StartSetPasswordFlow(ctx, resellerTgID)
	_ = bot.ProcessResellerStep(ctx, resellerTgID, "mypassword123")
	ack := sender.sentMessages[len(sender.sentMessages)-1]
	if !strings.Contains(ack, "Web panel password set") {
		t.Errorf("password setup ack missing: %s", ack)
	}
}

func TestAdminCreateClientAndManualCredit(t *testing.T) {
	adminTgID := int64(1001)
	inst := domain.Instance{
		ID:              1,
		Name:            "Main Bot",
		Type:            domain.InstanceTypeCustomer,
		AdminTelegramID: adminTgID,
		DefaultLang:     domain.LangEN,
		PanelURL:        "https://panel.example.com:2053",
	}
	bot, sender, walletRepo := setupTestBot(t, inst)
	ctx := context.Background()

	// 1. Admin Create Client directly in 3x-ui (P07)
	plan, _ := bot.GetPlan(1)
	err := bot.AdminCreateClient(ctx, adminTgID, 2002, plan, 1)
	if err != nil {
		t.Fatalf("AdminCreateClient failed: %v", err)
	}
	if len(sender.sentMessages) == 0 || !strings.Contains(sender.sentMessages[0], "Client created for user 2002") {
		t.Errorf("admin confirmation missing: %+v", sender.sentMessages)
	}

	// 2. Manual Credit: Amount on first line, optional admin message on next line (P10)
	sender.sentMessages = nil
	_ = bot.StartManualCreditFlow(ctx, adminTgID, "single", 2002)

	input := "100000\nCompensation bonus for network outage"
	_ = bot.ProcessManualCredit(ctx, adminTgID, input)

	// User 2002 should receive notification with amount and message (sentMessages[0] is admin prompt, [1] is user notification, [2] is admin ack)
	if len(sender.sentMessages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(sender.sentMessages))
	}
	userNotif := sender.sentMessages[1]
	if !strings.Contains(userNotif, "Compensation bonus for network outage") || !strings.Contains(userNotif, "100 thousand toman") {
		t.Errorf("manual credit notification missing required format: %s", userNotif)
	}
	_ = walletRepo
}
