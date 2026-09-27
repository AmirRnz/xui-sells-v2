package pricing

import (
	"errors"
	"fmt"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

var (
	ErrInvalidUserCount     = errors.New("user count must be at least 1")
	ErrUserCountExceedsMax  = errors.New("user count exceeds plan maximum allowed")
	ErrInvalidDuration      = errors.New("duration is not supported for this plan")
	ErrInvalidUpgradeUsers  = errors.New("new user count must be greater than current user count")
	ErrInvalidRemainingDays = errors.New("remaining days cannot be negative")
)

// Quote contains the detailed pricing calculation for a subscription purchase or renewal.
type Quote struct {
	PlanID         int64       `json:"plan_id"`
	UserCount      int         `json:"user_count"`
	ExtraUsers     int         `json:"extra_users"`
	DurationMonths int         `json:"duration_months"`
	DiscountPct    int         `json:"discount_pct"`
	MonthlyPrice   money.Money `json:"monthly_price"`
	Total          money.Money `json:"total"`
}

// CalculateQuote calculates the exact pricing quote according to specification D02.
// Formula:
//
//	effective_users = max(users, base_users)
//	extra_users = effective_users - base_users
//	monthly_price = base_price + (extra_users * extra_user_price)
//	discount_pct = discounts[duration_months] (default 0%)
//	total = round(monthly_price * duration_months * (100 - discount_pct) / 100)
func CalculateQuote(plan domain.Plan, userCount int, durationMonths int) (Quote, error) {
	if userCount < 1 {
		return Quote{}, ErrInvalidUserCount
	}
	if plan.MaxUsers > 0 && userCount > plan.MaxUsers {
		return Quote{}, fmt.Errorf("%w: requested %d, maximum %d", ErrUserCountExceedsMax, userCount, plan.MaxUsers)
	}

	if len(plan.Durations) > 0 {
		allowed := false
		for _, d := range plan.Durations {
			if d == durationMonths {
				allowed = true
				break
			}
		}
		if !allowed {
			return Quote{}, fmt.Errorf("%w: %d months", ErrInvalidDuration, durationMonths)
		}
	} else if durationMonths <= 0 {
		return Quote{}, fmt.Errorf("%w: %d months", ErrInvalidDuration, durationMonths)
	}

	extraUsers := 0
	if userCount > plan.BaseUsers {
		extraUsers = userCount - plan.BaseUsers
	}

	monthlyAmount := plan.BasePrice.Amount + int64(extraUsers)*plan.ExtraUserPrice.Amount
	monthlyPrice := money.New(monthlyAmount, plan.BasePrice.Currency)

	discountPct := 0
	if plan.Discounts != nil {
		if d, ok := plan.Discounts[durationMonths]; ok {
			discountPct = d
		}
	}
	if discountPct < 0 {
		discountPct = 0
	} else if discountPct > 100 {
		discountPct = 100
	}

	// Exact integer rounding for (monthlyPrice * durationMonths * (100 - discountPct)) / 100
	raw := monthlyPrice.Amount * int64(durationMonths) * int64(100-discountPct)
	totalAmount := (raw + 50) / 100
	total := money.New(totalAmount, plan.BasePrice.Currency)

	return Quote{
		PlanID:         plan.ID,
		UserCount:      userCount,
		ExtraUsers:     extraUsers,
		DurationMonths: durationMonths,
		DiscountPct:    discountPct,
		MonthlyPrice:   monthlyPrice,
		Total:          total,
	}, nil
}

// CalculateProratedUpgrade calculates the cost to increase user count for the remaining active days in the current period.
// Formula (D02): delta_users * extra_user_price * (remaining_days / 30)
func CalculateProratedUpgrade(plan domain.Plan, currentUsers int, newUsers int, remainingDays int) (money.Money, error) {
	if newUsers <= currentUsers {
		return money.Money{}, ErrInvalidUpgradeUsers
	}
	if plan.MaxUsers > 0 && newUsers > plan.MaxUsers {
		return money.Money{}, fmt.Errorf("%w: requested %d, maximum %d", ErrUserCountExceedsMax, newUsers, plan.MaxUsers)
	}
	if remainingDays < 0 {
		return money.Money{}, ErrInvalidRemainingDays
	}
	if remainingDays == 0 {
		return money.New(0, plan.ExtraUserPrice.Currency), nil
	}

	deltaUsers := int64(newUsers - currentUsers)
	raw := deltaUsers * plan.ExtraUserPrice.Amount * int64(remainingDays)
	// Rounding with + 15 for division by 30
	proratedAmount := (raw + 15) / 30

	return money.New(proratedAmount, plan.ExtraUserPrice.Currency), nil
}
