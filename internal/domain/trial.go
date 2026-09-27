package domain

import "time"

// TierVariant represents tier-specific trial variants.
type TierVariant string

const (
	TierVariantFree     TierVariant = "free"
	TierVariantPro      TierVariant = "pro"
	TierVariantUltimate TierVariant = "ultimate"
)

// TrialRecord tracks a free trial issued to a user.
type TrialRecord struct {
	ID         int64       `json:"id"`
	InstanceID int64       `json:"instance_id"`
	UserID     int64       `json:"user_id"`
	TelegramID int64       `json:"telegram_id"`
	PlanID     int64       `json:"plan_id"`
	ServiceID  int64       `json:"service_id"`
	Variant    TierVariant `json:"variant"`
	CreatedAt  time.Time   `json:"created_at"`
}
