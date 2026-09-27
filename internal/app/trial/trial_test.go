package trial_test

import (
	"context"
	"testing"
	"time"

	"xui-sells-v2/internal/app/trial"
	"xui-sells-v2/internal/domain"
)

type mockTrialRepo struct {
	records []domain.TrialRecord
}

func (m *mockTrialRepo) GetLastTrialTime(ctx context.Context, instanceID string, userTgID int64, planID int64) (*time.Time, error) {
	var latest *time.Time
	for _, r := range m.records {
		if r.TelegramID == userTgID && r.PlanID == planID {
			t := r.CreatedAt
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}
	return latest, nil
}

func (m *mockTrialRepo) CountTrialsInWindow(ctx context.Context, instanceID string, userTgID int64, planID int64, windowStart time.Time) (int, error) {
	count := 0
	for _, r := range m.records {
		if r.TelegramID == userTgID && r.PlanID == planID && (r.CreatedAt.After(windowStart) || r.CreatedAt.Equal(windowStart)) {
			count++
		}
	}
	return count, nil
}

func (m *mockTrialRepo) CountDailyResellerTrials(ctx context.Context, instanceID string, since time.Time) (int, error) {
	count := 0
	for _, r := range m.records {
		if r.CreatedAt.After(since) || r.CreatedAt.Equal(since) {
			count++
		}
	}
	return count, nil
}

func (m *mockTrialRepo) RecordTrial(ctx context.Context, instanceID string, record domain.TrialRecord) error {
	m.records = append(m.records, record)
	return nil
}

func TestStrictPlanIndependentTrials(t *testing.T) {
	repo := &mockTrialRepo{}
	svc := trial.NewService(repo)
	ctx := context.Background()

	planA := domain.Plan{
		ID:   1,
		Name: "Plan A",
		TrialConfig: domain.TrialConfig{
			TrafficBytes:    1073741824, // 1 GB
			DurationSeconds: 86400,
			CooldownSeconds: 3600, // 1 hour cooldown
			WindowSeconds:   86400,
			MaxPerWindow:    1, // Only 1 per day
		},
	}

	planB := domain.Plan{
		ID:   2,
		Name: "Plan B",
		TrialConfig: domain.TrialConfig{
			TrafficBytes:    1073741824,
			DurationSeconds: 86400,
			CooldownSeconds: 3600,
			WindowSeconds:   86400,
			MaxPerWindow:    1,
		},
	}

	userTgID := int64(123456)

	// User consumes trial on Plan A
	if err := svc.ConsumeTrial(ctx, "inst1", userTgID, planA, ""); err != nil {
		t.Fatalf("first trial on Plan A failed: %v", err)
	}

	// Immediate second trial on Plan A should fail due to cooldown and window quota
	if err := svc.CheckEligibility(ctx, "inst1", userTgID, planA, ""); err == nil {
		t.Fatalf("expected trial on Plan A to fail due to cooldown/quota")
	}

	// STRICT PER-PLAN INDEPENDENCE: User MUST still be eligible for Plan B!
	if err := svc.CheckEligibility(ctx, "inst1", userTgID, planB, ""); err != nil {
		t.Fatalf("trial on Plan B should be eligible despite consuming Plan A: %v", err)
	}

	// User consumes trial on Plan B
	if err := svc.ConsumeTrial(ctx, "inst1", userTgID, planB, ""); err != nil {
		t.Fatalf("trial on Plan B failed: %v", err)
	}
}

func TestTrialCooldownAndWindowQuota(t *testing.T) {
	repo := &mockTrialRepo{}
	svc := trial.NewService(repo)
	ctx := context.Background()

	plan := domain.Plan{
		ID: 10,
		TrialConfig: domain.TrialConfig{
			TrafficBytes:    2147483648,
			DurationSeconds: 86400,
			CooldownSeconds: 7200,   // 2 hours
			WindowSeconds:   604800, // 7 days
			MaxPerWindow:    2,      // Max 2 trials per week
		},
	}

	userTgID := int64(789)

	// Consume 1st trial
	if err := svc.ConsumeTrial(ctx, "inst1", userTgID, plan, ""); err != nil {
		t.Fatalf("failed to consume 1st trial: %v", err)
	}

	// Cooldown active immediately
	if err := svc.CheckEligibility(ctx, "inst1", userTgID, plan, ""); err == nil {
		t.Fatalf("expected error due to active cooldown")
	}

	// Fast forward past cooldown (e.g. 3 hours ago)
	repo.records[0].CreatedAt = time.Now().Add(-3 * time.Hour)

	// Should now be eligible for 2nd trial (within 2 max per week quota)
	if err := svc.CheckEligibility(ctx, "inst1", userTgID, plan, ""); err != nil {
		t.Fatalf("expected eligibility after cooldown elapsed: %v", err)
	}

	// Consume 2nd trial
	if err := svc.ConsumeTrial(ctx, "inst1", userTgID, plan, ""); err != nil {
		t.Fatalf("failed to consume 2nd trial: %v", err)
	}

	// Fast forward past cooldown for 2nd trial
	repo.records[1].CreatedAt = time.Now().Add(-3 * time.Hour)

	// Now window quota (2 per week) is exhausted!
	if err := svc.CheckEligibility(ctx, "inst1", userTgID, plan, ""); err == nil {
		t.Fatalf("expected error due to window quota exhaustion")
	}
}

func TestResellerTierDailyCap(t *testing.T) {
	repo := &mockTrialRepo{}
	svc := trial.NewService(repo)
	ctx := context.Background()

	plan := domain.Plan{
		ID: 20,
		TierTrialConfigs: domain.TierTrialConfigs{
			domain.TierVariantFree: domain.TrialConfig{
				TrafficBytes:    1073741824,
				DurationSeconds: 86400,
			},
		},
	}

	// Free reseller tier has a daily cap of 10 trials
	for i := 0; i < 10; i++ {
		err := svc.ConsumeTrial(ctx, "reseller_inst", int64(1000+i), plan, domain.TierVariantFree)
		if err != nil {
			t.Fatalf("trial %d should succeed: %v", i+1, err)
		}
	}

	// 11th trial should fail due to daily cap
	err := svc.CheckEligibility(ctx, "reseller_inst", 9999, plan, domain.TierVariantFree)
	if err == nil {
		t.Fatalf("expected error exceeding Free tier daily cap of 10")
	}
}
