package telegram

import (
	"context"
	"sync"

	"xui-sells-v2/internal/app/broadcast"
	"xui-sells-v2/internal/app/entitlement"
	"xui-sells-v2/internal/app/order"
	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/app/stats"
	"xui-sells-v2/internal/app/ticket"
	"xui-sells-v2/internal/app/trial"
	"xui-sells-v2/internal/app/wallet"
	"xui-sells-v2/internal/domain"
)

// UserStep represents the current position in an interactive bot wizard.
type UserStep string

const (
	StepNone                 UserStep = ""
	StepResellerSetGroupName UserStep = "reseller_set_group_name"
	StepResellerSetPassword  UserStep = "reseller_set_password"
	StepResellerAddChildBot  UserStep = "reseller_add_child_bot"
	StepBuySelectDuration    UserStep = "buy_select_duration"
	StepBuyEnterUsers        UserStep = "buy_enter_users"
	StepBuyEnterName         UserStep = "buy_enter_name"
	StepBuySelectPayment     UserStep = "buy_select_payment"
	StepBuySubmitReceipt     UserStep = "buy_submit_receipt"
	StepAdminManualCredit    UserStep = "admin_manual_credit"
	StepAdminBroadcast       UserStep = "admin_broadcast"
)

// WizardState tracks multi-step flow state for a user.
type WizardState struct {
	Step           UserStep
	SelectedPlanID int64
	DurationMonths int
	UserCount      int
	SubName        string
	PendingOrderID int64
	AdminCreditScope string // "single", "selected", "all"
	AdminCreditTarget int64
	BroadcastAudience broadcast.AudienceType
}

// Sender abstracts message and media dispatch to allow unit testing without a Telegram network connection.
type Sender interface {
	SendMessage(ctx context.Context, chatID int64, text string, replyMarkup any) error
	SendPhoto(ctx context.Context, chatID int64, photo []byte, caption string, replyMarkup any) error
}

// BotDependencies holds references to shared application core services.
type BotDependencies struct {
	TrialSvc       *trial.Service
	WalletSvc      *wallet.Service
	ProvSvc        *provisioning.Service
	OrderSvc       *order.Service
	EntitlementSvc *entitlement.Resolver
	TicketSvc      *ticket.Service
	StatsSvc       *stats.Service
	BroadcastSvc   *broadcast.Service
	Sender         Sender
	Supervisor     *Supervisor
}

// BotInstance represents a running bot runtime attached to an instance.
type BotInstance struct {
	Instance domain.Instance
	Deps     BotDependencies

	mu     sync.RWMutex
	states map[int64]*WizardState
	users  map[int64]*domain.User
	plans  map[int64]domain.Plan
}

// NewBotInstance constructs a bot runtime.
func NewBotInstance(inst domain.Instance, deps BotDependencies) *BotInstance {
	return &BotInstance{
		Instance: inst,
		Deps:     deps,
		states:   make(map[int64]*WizardState),
		users:    make(map[int64]*domain.User),
		plans:    make(map[int64]domain.Plan),
	}
}

// GetState returns the current wizard state for a user.
func (b *BotInstance) GetState(tgID int64) *WizardState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	s, ok := b.states[tgID]
	if !ok {
		return &WizardState{Step: StepNone}
	}
	cp := *s
	return &cp
}

// SetState updates the user's wizard state.
func (b *BotInstance) SetState(tgID int64, s *WizardState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if s == nil {
		delete(b.states, tgID)
	} else {
		cp := *s
		b.states[tgID] = &cp
	}
}

// ClearState resets a user's active wizard.
func (b *BotInstance) ClearState(tgID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.states, tgID)
}

// GetUser retrieves or caches a user in this instance.
func (b *BotInstance) GetUser(tgID int64) *domain.User {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if u, ok := b.users[tgID]; ok {
		cp := *u
		return &cp
	}
	return nil
}

// SaveUser stores or updates a user.
func (b *BotInstance) SaveUser(u domain.User) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.users[u.TelegramID] = &u
}

// SetPlans configures available plans in the instance.
func (b *BotInstance) SetPlans(plans []domain.Plan) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.plans = make(map[int64]domain.Plan)
	for _, p := range plans {
		b.plans[p.ID] = p
	}
}

// GetPlan retrieves a plan by ID.
func (b *BotInstance) GetPlan(id int64) (domain.Plan, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	p, ok := b.plans[id]
	return p, ok
}

// ListPlans returns all active plans.
func (b *BotInstance) ListPlans() []domain.Plan {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var res []domain.Plan
	for _, p := range b.plans {
		if p.IsActive {
			res = append(res, p)
		}
	}
	return res
}
