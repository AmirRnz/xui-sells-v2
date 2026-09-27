package entitlement

import (
	"time"

	"xui-sells-v2/internal/domain"
)

// TierLimits describes the quotas, refund windows, and feature gates for a reseller tier.
type TierLimits struct {
	Tier                 domain.ResellerTier
	DailyTrialCap        int
	RefundWindowDays     int
	CanCreateChildBot    bool
	CanAccessWebPanel    bool
	AllowedTrialVariants []domain.TierVariant
}

// Resolver calculates reseller privileges, plan entitlements, and feature gating.
type Resolver struct{}

// NewResolver creates a new entitlement resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

// ResolveEffectiveTier returns the reseller's current tier, automatically reverting to Free if expired (D07).
func (r *Resolver) ResolveEffectiveTier(profile domain.ResellerProfile) domain.ResellerTier {
	if profile.TierExpiresAt != nil && time.Now().After(*profile.TierExpiresAt) {
		return domain.ResellerTierFree
	}
	return profile.Tier
}

// GetTierLimits returns the configured privileges and limits for a tier (D07, P17, P18).
func (r *Resolver) GetTierLimits(tier domain.ResellerTier) TierLimits {
	switch tier {
	case domain.ResellerTierPro:
		return TierLimits{
			Tier:              domain.ResellerTierPro,
			DailyTrialCap:     25,
			RefundWindowDays:  2,
			CanCreateChildBot: true,
			CanAccessWebPanel: false,
			AllowedTrialVariants: []domain.TierVariant{
				domain.TierVariantFree,
				domain.TierVariantPro,
			},
		}

	case domain.ResellerTierUltimate:
		return TierLimits{
			Tier:              domain.ResellerTierUltimate,
			DailyTrialCap:     100,
			RefundWindowDays:  3,
			CanCreateChildBot: true,
			CanAccessWebPanel: true,
			AllowedTrialVariants: []domain.TierVariant{
				domain.TierVariantFree,
				domain.TierVariantPro,
				domain.TierVariantUltimate,
			},
		}

	case domain.ResellerTierFree:
		fallthrough
	default:
		return TierLimits{
			Tier:              domain.ResellerTierFree,
			DailyTrialCap:     10,
			RefundWindowDays:  1,
			CanCreateChildBot: false,
			CanAccessWebPanel: false,
			AllowedTrialVariants: []domain.TierVariant{
				domain.TierVariantFree,
			},
		}
	}
}

// CanAccessPlan returns whether a reseller with a given tier can access or sell a plan.
func (r *Resolver) CanAccessPlan(tier domain.ResellerTier, minPlanTier domain.ResellerTier) bool {
	return tier >= minPlanTier
}

// CanCreateChildBot checks if the reseller tier entitles creation of child bots (P19).
func (r *Resolver) CanCreateChildBot(tier domain.ResellerTier) bool {
	return r.GetTierLimits(tier).CanCreateChildBot
}

// CanAccessWebPanel checks if the reseller tier entitles access to the web panel (P21).
func (r *Resolver) CanAccessWebPanel(tier domain.ResellerTier) bool {
	return r.GetTierLimits(tier).CanAccessWebPanel
}

// IsTrialVariantAllowed checks whether a tier can issue a specific trial variant (P18).
func (r *Resolver) IsTrialVariantAllowed(tier domain.ResellerTier, variant domain.TierVariant) bool {
	allowed := r.GetTierLimits(tier).AllowedTrialVariants
	for _, v := range allowed {
		if v == variant {
			return true
		}
	}
	return false
}

// GetAllowedTrialVariants returns all trial variants permitted for a tier.
func (r *Resolver) GetAllowedTrialVariants(tier domain.ResellerTier) []domain.TierVariant {
	return r.GetTierLimits(tier).AllowedTrialVariants
}

// GetRefundWindowDays returns the maximum allowable refund window for a tier.
func (r *Resolver) GetRefundWindowDays(tier domain.ResellerTier) int {
	return r.GetTierLimits(tier).RefundWindowDays
}

// GetDailyTrialCap returns the aggregate daily trial quota for a tier.
func (r *Resolver) GetDailyTrialCap(tier domain.ResellerTier) int {
	return r.GetTierLimits(tier).DailyTrialCap
}
