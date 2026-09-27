package trial

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"xui-sells-v2/internal/domain"
)

var (
	ErrTrialNotAvailable       = errors.New("free trials are not available for this plan or tier")
	ErrTrialCooldownActive      = errors.New("trial cooldown period is still active for this plan")
	ErrTrialQuotaExceeded       = errors.New("trial quota exceeded for the current rolling window on this plan")
	ErrResellerDailyCapExceeded = errors.New("reseller daily trial quota has been reached")
)

// DailyCapForTier returns the aggregate daily trial limit for a reseller tier.
// Free: 10/day, Pro: 25/day, Ultimate: 100/day.
func DailyCapForTier(tier domain.TierVariant) int {
	switch tier {
	case domain.TierVariantFree:
		return 10
	case domain.TierVariantPro:
		return 25
	case domain.TierVariantUltimate:
		return 100
	default:
		return 10
	}
}

// TrialRepository abstracts persistence for trial history and quotas.
type TrialRepository interface {
	GetLastTrialTime(ctx context.Context, instanceID string, userTgID int64, planID int64) (*time.Time, error)
	CountTrialsInWindow(ctx context.Context, instanceID string, userTgID int64, planID int64, windowStart time.Time) (int, error)
	CountDailyResellerTrials(ctx context.Context, instanceID string, since time.Time) (int, error)
	RecordTrial(ctx context.Context, instanceID string, record domain.TrialRecord) error
}

// Service manages trial eligibility and consumption.
type Service struct {
	repo TrialRepository
}

// NewService creates a new trial service.
func NewService(repo TrialRepository) *Service {
	return &Service{repo: repo}
}

// CheckEligibility evaluates whether a user is eligible for a free trial on a specific plan.
// In accordance with D01 & anti_duplication_log:
// 1. Evaluates cooldown and rolling window quota conjunctively.
// 2. Enforces STRICT per-plan independence by scoping queries to plan.ID.
// 3. Evaluates reseller aggregate daily trial cap across all plans when tier is specified.
func (s *Service) CheckEligibility(ctx context.Context, instanceID string, userTgID int64, plan domain.Plan, tier domain.TierVariant) error {
	var cfg domain.TrialConfig

	if tier != "" {
		if len(plan.TierTrialConfigs) > 0 {
			var ok bool
			cfg, ok = plan.TierTrialConfigs[tier]
			if !ok || cfg.TrafficBytes <= 0 {
				return fmt.Errorf("%w: tier %s", ErrTrialNotAvailable, tier)
			}
		} else {
			cfg = plan.TrialConfig
		}
	} else {
		cfg = plan.TrialConfig
	}

	if cfg.TrafficBytes <= 0 {
		return ErrTrialNotAvailable
	}

	now := time.Now()

	// 1. Check reseller daily cap across all plans if tier is specified
	if tier != "" {
		dailyCap := DailyCapForTier(tier)
		since := now.Add(-24 * time.Hour)
		dailyCount, err := s.repo.CountDailyResellerTrials(ctx, instanceID, since)
		if err != nil {
			return fmt.Errorf("failed to check reseller daily trial count: %w", err)
		}
		if dailyCount >= dailyCap {
			return fmt.Errorf("%w: %d/%d used in 24h", ErrResellerDailyCapExceeded, dailyCount, dailyCap)
		}
	}

	// 2. Conjunctive Check A: Cooldown (strictly per plan.ID)
	if cfg.CooldownSeconds > 0 {
		lastTrialTime, err := s.repo.GetLastTrialTime(ctx, instanceID, userTgID, plan.ID)
		if err != nil {
			return fmt.Errorf("failed to check trial cooldown: %w", err)
		}
		if lastTrialTime != nil {
			cooldownDuration := time.Duration(cfg.CooldownSeconds) * time.Second
			if now.Sub(*lastTrialTime) < cooldownDuration {
				return fmt.Errorf("%w: cooldown %ds remaining", ErrTrialCooldownActive, int((cooldownDuration - now.Sub(*lastTrialTime)).Seconds()))
			}
		}
	}

	// 3. Conjunctive Check B: Rolling Window Quota (strictly per plan.ID)
	if cfg.MaxPerWindow > 0 && cfg.WindowSeconds > 0 {
		windowStart := now.Add(-time.Duration(cfg.WindowSeconds) * time.Second)
		count, err := s.repo.CountTrialsInWindow(ctx, instanceID, userTgID, plan.ID, windowStart)
		if err != nil {
			return fmt.Errorf("failed to count trials in window: %w", err)
		}
		if count >= cfg.MaxPerWindow {
			return fmt.Errorf("%w: reached %d of %d within %ds", ErrTrialQuotaExceeded, count, cfg.MaxPerWindow, cfg.WindowSeconds)
		}
	}

	return nil
}

// ConsumeTrial verifies eligibility and records the trial.
func (s *Service) ConsumeTrial(ctx context.Context, instanceID string, userTgID int64, plan domain.Plan, tier domain.TierVariant) error {
	if err := s.CheckEligibility(ctx, instanceID, userTgID, plan, tier); err != nil {
		return err
	}

	instID, _ := strconv.ParseInt(instanceID, 10, 64)
	record := domain.TrialRecord{
		InstanceID: instID,
		TelegramID: userTgID,
		PlanID:     plan.ID,
		Variant:    tier,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.RecordTrial(ctx, instanceID, record); err != nil {
		return fmt.Errorf("failed to record trial: %w", err)
	}

	return nil
}
