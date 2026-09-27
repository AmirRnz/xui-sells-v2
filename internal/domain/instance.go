package domain

import (
	"time"

	"xui-sells-v2/internal/domain/money"
)

// InstanceType defines the nature of the bot instance.
type InstanceType string

const (
	InstanceTypeCustomer      InstanceType = "customer"
	InstanceTypeReseller      InstanceType = "reseller"
	InstanceTypeChildCustomer InstanceType = "child_customer"
)

// ResellerTier represents the membership level of a reseller.
type ResellerTier int

const (
	ResellerTierFree     ResellerTier = 0
	ResellerTierPro      ResellerTier = 1
	ResellerTierUltimate ResellerTier = 2
)

func (t ResellerTier) String() string {
	switch t {
	case ResellerTierFree:
		return "free"
	case ResellerTierPro:
		return "pro"
	case ResellerTierUltimate:
		return "ultimate"
	default:
		return "unknown"
	}
}

// ResellerProfile stores extended metadata and tier memberships for reseller instances.
type ResellerProfile struct {
	ID                   int64        `json:"id"`
	InstanceID           int64        `json:"instance_id"`
	TelegramID           int64        `json:"telegram_id"`
	Tier                 ResellerTier `json:"tier"`
	TierExpiresAt        *time.Time   `json:"tier_expires_at,omitempty"`
	DailyTrialCap        int          `json:"daily_trial_cap"`
	WebPanelPasswordHash string       `json:"web_panel_password_hash,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// Instance represents a bot instance (customer bot, reseller bot, or child bot).
type Instance struct {
	ID                 int64          `json:"id"`
	Name               string         `json:"name"`
	Type               InstanceType   `json:"type"`
	BotToken           string         `json:"bot_token"`
	DefaultLang        Language       `json:"default_lang"`
	PanelURL           string         `json:"panel_url"`
	PanelAPIKey        string         `json:"panel_api_key"`
	ParentInstanceID   *int64         `json:"parent_instance_id,omitempty"`
	AdminTelegramID    int64          `json:"admin_telegram_id"`
	GroupName          string         `json:"group_name"`
	CardNumber         string         `json:"card_number"`
	CardHolder         string         `json:"card_holder"`
	MinTopup           money.Money    `json:"min_topup"`
	Currency           money.Currency `json:"currency"`
	SupportTelegramID  string         `json:"support_telegram_id"`
	ReferralPercentage int            `json:"referral_percentage"`
	RefundWindowDays   int            `json:"refund_window_days"`
	IsActive           bool           `json:"is_active"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// IsChild returns true if the instance is a reseller-owned child customer bot.
func (i Instance) IsChild() bool {
	return i.Type == InstanceTypeChildCustomer
}

// IsReseller returns true if the instance is a reseller bot.
func (i Instance) IsReseller() bool {
	return i.Type == InstanceTypeReseller
}

// IsCustomer returns true if the instance is a standalone ordinary customer bot.
func (i Instance) IsCustomer() bool {
	return i.Type == InstanceTypeCustomer
}
