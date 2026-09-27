package entitlement_test

import (
	"testing"
	"time"

	"xui-sells-v2/internal/app/entitlement"
	"xui-sells-v2/internal/domain"
)

func TestTierLimitsAndFeatureGates(t *testing.T) {
	resolver := entitlement.NewResolver()

	// 1. Free Tier (0)
	free := resolver.GetTierLimits(domain.ResellerTierFree)
	if free.DailyTrialCap != 10 || free.RefundWindowDays != 1 {
		t.Errorf("unexpected free tier limits: %+v", free)
	}
	if free.CanCreateChildBot || free.CanAccessWebPanel {
		t.Errorf("free tier should not have child bot or web panel access")
	}
	if len(free.AllowedTrialVariants) != 1 || free.AllowedTrialVariants[0] != domain.TierVariantFree {
		t.Errorf("free tier should only allow free variant")
	}

	// 2. Pro Tier (1)
	pro := resolver.GetTierLimits(domain.ResellerTierPro)
	if pro.DailyTrialCap != 25 || pro.RefundWindowDays != 2 {
		t.Errorf("unexpected pro tier limits: %+v", pro)
	}
	if !pro.CanCreateChildBot || pro.CanAccessWebPanel {
		t.Errorf("pro tier should have child bot access, but not web panel access")
	}
	if !resolver.IsTrialVariantAllowed(domain.ResellerTierPro, domain.TierVariantFree) ||
		!resolver.IsTrialVariantAllowed(domain.ResellerTierPro, domain.TierVariantPro) ||
		resolver.IsTrialVariantAllowed(domain.ResellerTierPro, domain.TierVariantUltimate) {
		t.Errorf("pro tier should allow free and pro, but not ultimate")
	}

	// 3. Ultimate Tier (2)
	ultimate := resolver.GetTierLimits(domain.ResellerTierUltimate)
	if ultimate.DailyTrialCap != 100 || ultimate.RefundWindowDays != 3 {
		t.Errorf("unexpected ultimate tier limits: %+v", ultimate)
	}
	if !ultimate.CanCreateChildBot || !ultimate.CanAccessWebPanel {
		t.Errorf("ultimate tier must have both child bot and web panel access")
	}
	if len(ultimate.AllowedTrialVariants) != 3 {
		t.Errorf("ultimate tier must allow all 3 trial variants")
	}
}

func TestPlanAccessEntitlement(t *testing.T) {
	resolver := entitlement.NewResolver()

	// Free cannot access Pro or Ultimate plans
	if resolver.CanAccessPlan(domain.ResellerTierFree, domain.ResellerTierPro) {
		t.Errorf("free tier should not access pro plans")
	}
	if resolver.CanAccessPlan(domain.ResellerTierFree, domain.ResellerTierUltimate) {
		t.Errorf("free tier should not access ultimate plans")
	}

	// Pro can access Free and Pro plans
	if !resolver.CanAccessPlan(domain.ResellerTierPro, domain.ResellerTierFree) ||
		!resolver.CanAccessPlan(domain.ResellerTierPro, domain.ResellerTierPro) {
		t.Errorf("pro tier should access free and pro plans")
	}
	if resolver.CanAccessPlan(domain.ResellerTierPro, domain.ResellerTierUltimate) {
		t.Errorf("pro tier should not access ultimate plans")
	}

	// Ultimate can access all plans
	if !resolver.CanAccessPlan(domain.ResellerTierUltimate, domain.ResellerTierFree) ||
		!resolver.CanAccessPlan(domain.ResellerTierUltimate, domain.ResellerTierPro) ||
		!resolver.CanAccessPlan(domain.ResellerTierUltimate, domain.ResellerTierUltimate) {
		t.Errorf("ultimate tier should access all plans")
	}
}

func TestResolveEffectiveTierExpiry(t *testing.T) {
	resolver := entitlement.NewResolver()

	// Active Pro Tier
	future := time.Now().Add(10 * 24 * time.Hour)
	activeProfile := domain.ResellerProfile{
		Tier:          domain.ResellerTierPro,
		TierExpiresAt: &future,
	}
	if resolver.ResolveEffectiveTier(activeProfile) != domain.ResellerTierPro {
		t.Errorf("active profile should retain pro tier")
	}

	// Expired Ultimate Tier -> Automatically reverts to Free (D07)
	past := time.Now().Add(-1 * time.Hour)
	expiredProfile := domain.ResellerProfile{
		Tier:          domain.ResellerTierUltimate,
		TierExpiresAt: &past,
	}
	if resolver.ResolveEffectiveTier(expiredProfile) != domain.ResellerTierFree {
		t.Errorf("expired profile must revert to free tier")
	}
}
