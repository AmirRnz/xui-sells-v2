package stats

import (
	"context"
	"fmt"

	"xui-sells-v2/internal/domain/money"
)

// TrafficChartPoint represents aggregated traffic usage per day.
type TrafficChartPoint struct {
	Date       string `json:"date"`
	UploadGB   int    `json:"upload_gb"`
	DownloadGB int    `json:"download_gb"`
	TotalGB    int    `json:"total_gb"`
}

// RevenueChartPoint represents aggregated sales per day or week.
type RevenueChartPoint struct {
	Date        string `json:"date"`
	Revenue     int64  `json:"revenue"`
	OrdersCount int    `json:"orders_count"`
}

// OperationalStats aggregates system metrics across users, services, orders, and ledger (P11).
type OperationalStats struct {
	RegisteredUsers  int                 `json:"registered_users"`
	NewUsers         int                 `json:"new_users"`
	ActiveServices   int                 `json:"active_services"`
	ExpiredServices  int                 `json:"expired_services"`
	TrialsIssued     int                 `json:"trials_issued"`
	ApprovedOrders   int                 `json:"approved_orders"`
	PendingApprovals int                 `json:"pending_approvals"`
	WalletFunding    money.Money         `json:"wallet_funding"`
	TotalRevenue     money.Money         `json:"total_revenue"`
	TotalRefunds     money.Money         `json:"total_refunds"`
	ReferralRewards  money.Money         `json:"referral_rewards"`
	TotalCustomers   int                 `json:"total_customers"`
	WalletBalance    int64               `json:"wallet_balance"`
	MonthlyRevenue   int64               `json:"monthly_revenue"`
	TrafficChartData []TrafficChartPoint `json:"traffic_chart_data"`
	RevenueChartData []RevenueChartPoint `json:"revenue_chart_data"`
}

// StatsRepository abstracts cross-table metric aggregation.
type StatsRepository interface {
	GetOperationalStats(ctx context.Context, instanceID int64) (*OperationalStats, error)
}

// Service provides metrics and analytics calculations.
type Service struct {
	repo StatsRepository
}

// NewService creates a new stats service.
func NewService(repo StatsRepository) *Service {
	return &Service{repo: repo}
}

// GetStats calculates and returns operational metrics for an instance.
func (s *Service) GetStats(ctx context.Context, instanceID int64) (*OperationalStats, error) {
	stats, err := s.repo.GetOperationalStats(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate operational stats: %w", err)
	}

	// Guarantee fields for Web Panel parity
	if stats.TotalCustomers == 0 && stats.RegisteredUsers > 0 {
		stats.TotalCustomers = stats.RegisteredUsers
	}
	if stats.MonthlyRevenue == 0 && stats.TotalRevenue.IsPositive() {
		stats.MonthlyRevenue = stats.TotalRevenue.Amount
	}

	return stats, nil
}
