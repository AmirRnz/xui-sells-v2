package stats_test

import (
	"context"
	"testing"

	"xui-sells-v2/internal/app/stats"
	"xui-sells-v2/internal/domain/money"
)

type mockStatsRepo struct {
	data *stats.OperationalStats
}

func (m *mockStatsRepo) GetOperationalStats(ctx context.Context, instanceID int64) (*stats.OperationalStats, error) {
	return m.data, nil
}

func TestStatsAggregation(t *testing.T) {
	mockData := &stats.OperationalStats{
		RegisteredUsers:  150,
		NewUsers:         25,
		ActiveServices:   80,
		ExpiredServices:  12,
		TrialsIssued:     40,
		ApprovedOrders:   95,
		PendingApprovals: 3,
		WalletFunding:    money.NewToman(5_000_000),
		TotalRevenue:     money.NewToman(18_000_000),
		TotalRefunds:     money.NewToman(400_000),
		ReferralRewards:  money.NewToman(1_800_000),
		TrafficChartData: []stats.TrafficChartPoint{
			{Date: "Mon", UploadGB: 10, DownloadGB: 40, TotalGB: 50},
		},
		RevenueChartData: []stats.RevenueChartPoint{
			{Date: "Week 1", Revenue: 4500000, OrdersCount: 20},
		},
	}

	svc := stats.NewService(&mockStatsRepo{data: mockData})
	res, err := svc.GetStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if res.ActiveServices != 80 || res.TotalRevenue.Amount != 18_000_000 {
		t.Errorf("mismatched stats: %+v", res)
	}
	if res.TotalCustomers != 150 {
		t.Errorf("expected TotalCustomers populated from RegisteredUsers, got %d", res.TotalCustomers)
	}
	if res.MonthlyRevenue != 18_000_000 {
		t.Errorf("expected MonthlyRevenue populated from TotalRevenue, got %d", res.MonthlyRevenue)
	}
}
