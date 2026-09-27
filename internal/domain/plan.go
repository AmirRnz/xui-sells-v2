package domain

import (
	"time"

	"xui-sells-v2/internal/domain/money"
)

// Duration represents a subscription duration in integer multiples of months.
// As defined in D02, 1 month is standardized as 30 calendar days (720 hours = 2,592,000 seconds).
type Duration int

// Months returns the duration in months.
func (d Duration) Months() int {
	return int(d)
}

// Days returns the duration in days (30 days per month).
func (d Duration) Days() int {
	return int(d) * 30
}

// Hours returns the duration in hours (720 hours per month).
func (d Duration) Hours() int {
	return int(d) * 720
}

// Seconds returns the duration in seconds (2,592,000 seconds per month).
func (d Duration) Seconds() int64 {
	return int64(d) * 2_592_000
}

// TimeDuration returns the duration as time.Duration.
func (d Duration) TimeDuration() time.Duration {
	return time.Duration(d.Seconds()) * time.Second
}

// Discounts maps duration in months to percentage discounts (e.g. 3 -> 10 for 10% off).
type Discounts map[int]int

// TrialConfig defines quota and cooldown constraints for free trials.
type TrialConfig struct {
	TrafficBytes    int64 `json:"traffic_bytes"`
	DurationSeconds int   `json:"duration_seconds"`
	CooldownSeconds int   `json:"cooldown_seconds"`
	WindowSeconds   int   `json:"window_seconds"`
	MaxPerWindow    int   `json:"max_per_window"`
}

// TierTrialConfigs stores trial configurations keyed by reseller tier variants (free, pro, ultimate).
type TierTrialConfigs map[TierVariant]TrialConfig

// Plan represents a VPN plan configuration.
type Plan struct {
	ID               int64            `json:"id"`
	InstanceID       int64            `json:"instance_id"`
	Name             string           `json:"name"`
	Description      string           `json:"description"` // Shown before purchase
	UsageNotes       string           `json:"usage_notes"`  // Shown after purchase
	InboundIDs       []int            `json:"inbound_ids"`  // Attached 3x-ui inbounds
	TrafficBytes     int64            `json:"traffic_bytes"`
	BasePrice        money.Money      `json:"base_price"`
	ExtraUserPrice   money.Money      `json:"extra_user_price"`
	BaseUsers        int              `json:"base_users"`
	MaxUsers         int              `json:"max_users"`
	Durations        []int            `json:"durations"` // Allowed months, e.g. [1, 2, 3, 6, 12]
	Discounts        Discounts        `json:"discounts"`
	TrialConfig      TrialConfig      `json:"trial_config"`
	TierTrialConfigs TierTrialConfigs `json:"tier_trial_configs"`
	MinResellerTier  ResellerTier     `json:"min_reseller_tier"`
	IsActive         bool             `json:"is_active"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}
