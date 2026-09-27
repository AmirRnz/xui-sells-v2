package pricing_test

import (
	"testing"

	"xui-sells-v2/internal/app/pricing"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

func samplePlan() domain.Plan {
	return domain.Plan{
		ID:             1,
		InstanceID:     1,
		Name:           "Standard VIP",
		BasePrice:      money.FromTomanInput(200), // 200,000 Toman
		ExtraUserPrice: money.NewToman(50_000),    // 50,000 Toman per extra user
		BaseUsers:      1,
		MaxUsers:       5,
		Durations:      []int{1, 2, 3, 5, 6, 12},
		Discounts: domain.Discounts{
			3:  10, // 10% off for 3 months
			6:  15, // 15% off for 6 months
			12: 20, // 20% off for 12 months
		},
	}
}

func TestCalculateQuoteStandard(t *testing.T) {
	plan := samplePlan()

	// 1 user, 1 month -> Base price 200,000 Toman
	quote, err := pricing.CalculateQuote(plan, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quote.MonthlyPrice.Amount != 200_000 {
		t.Errorf("expected monthly 200,000, got %d", quote.MonthlyPrice.Amount)
	}
	if quote.Total.Amount != 200_000 {
		t.Errorf("expected total 200,000, got %d", quote.Total.Amount)
	}
	if quote.ExtraUsers != 0 {
		t.Errorf("expected 0 extra users, got %d", quote.ExtraUsers)
	}
}

func TestCalculateQuoteFiveMonths(t *testing.T) {
	plan := samplePlan()

	// Example from P09 / D02: buying five months of 200k base price costs 1 million toman
	quote, err := pricing.CalculateQuote(plan, 1, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quote.Total.Amount != 1_000_000 {
		t.Errorf("expected total 1,000,000, got %d", quote.Total.Amount)
	}
	if quote.Total.Format("fa") != "1 میلیون تومان" {
		t.Errorf("expected '1 میلیون تومان', got %s", quote.Total.Format("fa"))
	}
}

func TestCalculateQuoteWithExtraUsersAndDiscount(t *testing.T) {
	plan := samplePlan()

	// 3 users (base 1, extra 2): monthly = 200,000 + 2*50,000 = 300,000
	// 3 months with 10% discount: 3 * 300,000 * 90 / 100 = 810,000
	quote, err := pricing.CalculateQuote(plan, 3, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quote.ExtraUsers != 2 {
		t.Errorf("expected 2 extra users, got %d", quote.ExtraUsers)
	}
	if quote.MonthlyPrice.Amount != 300_000 {
		t.Errorf("expected monthly 300,000, got %d", quote.MonthlyPrice.Amount)
	}
	if quote.DiscountPct != 10 {
		t.Errorf("expected 10%% discount, got %d", quote.DiscountPct)
	}
	if quote.Total.Amount != 810_000 {
		t.Errorf("expected total 810,000, got %d", quote.Total.Amount)
	}
}

func TestCalculateQuoteBoundsChecking(t *testing.T) {
	plan := samplePlan()

	// User count < 1
	if _, err := pricing.CalculateQuote(plan, 0, 1); err != pricing.ErrInvalidUserCount {
		t.Errorf("expected ErrInvalidUserCount for 0 users, got %v", err)
	}

	// User count > max users (max is 5)
	if _, err := pricing.CalculateQuote(plan, 6, 1); err == nil {
		t.Errorf("expected error for user count exceeding max")
	}

	// Duration not in plan.Durations
	if _, err := pricing.CalculateQuote(plan, 1, 4); err == nil {
		t.Errorf("expected error for unsupported duration")
	}
}

func TestCalculateProratedUpgrade(t *testing.T) {
	plan := samplePlan()
	// extraUserPrice is 50,000 Toman

	// 1 user -> 2 users (delta 1 user), 15 days remaining:
	// 1 * 50,000 * 15 / 30 = 25,000 Toman
	cost, err := pricing.CalculateProratedUpgrade(plan, 1, 2, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cost.Amount != 25_000 {
		t.Errorf("expected prorated 25,000, got %d", cost.Amount)
	}

	// 0 days remaining -> 0
	costZero, err := pricing.CalculateProratedUpgrade(plan, 1, 2, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if costZero.Amount != 0 {
		t.Errorf("expected 0 for 0 remaining days, got %d", costZero.Amount)
	}

	// Upgrade count <= current count -> error
	if _, err := pricing.CalculateProratedUpgrade(plan, 2, 2, 10); err != pricing.ErrInvalidUpgradeUsers {
		t.Errorf("expected ErrInvalidUpgradeUsers for equal count, got %v", err)
	}
	if _, err := pricing.CalculateProratedUpgrade(plan, 3, 2, 10); err != pricing.ErrInvalidUpgradeUsers {
		t.Errorf("expected ErrInvalidUpgradeUsers for decreasing count, got %v", err)
	}

	// Negative remaining days -> error
	if _, err := pricing.CalculateProratedUpgrade(plan, 1, 2, -5); err != pricing.ErrInvalidRemainingDays {
		t.Errorf("expected ErrInvalidRemainingDays, got %v", err)
	}

	// Exceeding plan max users
	if _, err := pricing.CalculateProratedUpgrade(plan, 1, 6, 15); err == nil {
		t.Errorf("expected error when upgrade exceeds MaxUsers")
	}
}
