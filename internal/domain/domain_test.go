package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

func TestInstanceDomain(t *testing.T) {
	inst := domain.Instance{
		ID:          1,
		Name:        "Test Bot",
		Type:        domain.InstanceTypeCustomer,
		DefaultLang: domain.LangFA,
		Currency:    money.CurrencyToman,
		MinTopup:    money.NewToman(50_000),
	}

	if !inst.IsCustomer() || inst.IsChild() || inst.IsReseller() {
		t.Fatalf("expected customer instance")
	}

	child := domain.Instance{
		ID:   2,
		Type: domain.InstanceTypeChildCustomer,
	}
	if !child.IsChild() {
		t.Fatalf("expected child instance")
	}

	reseller := domain.Instance{
		ID:   3,
		Type: domain.InstanceTypeReseller,
	}
	if !reseller.IsReseller() {
		t.Fatalf("expected reseller instance")
	}
}

func TestPlanDuration(t *testing.T) {
	d := domain.Duration(3)
	if d.Months() != 3 {
		t.Fatalf("expected 3 months, got %d", d.Months())
	}
	if d.Days() != 90 {
		t.Fatalf("expected 90 days, got %d", d.Days())
	}
	if d.Hours() != 2160 {
		t.Fatalf("expected 2160 hours, got %d", d.Hours())
	}
	if d.Seconds() != 7_776_000 {
		t.Fatalf("expected 7776000 seconds, got %d", d.Seconds())
	}
	if d.TimeDuration() != time.Duration(7_776_000)*time.Second {
		t.Fatalf("expected time duration mismatch")
	}
}

func TestServiceDomain(t *testing.T) {
	now := time.Now()
	svc := domain.Service{
		ID:          1,
		ClientEmail: "user_test",
		ExpiryTime:  now.Add(1 * time.Hour),
		Status:      domain.ServiceStatusActive,
	}

	if svc.IsExpired() {
		t.Fatalf("service should not be expired")
	}
	if !svc.IsActive() {
		t.Fatalf("service should be active")
	}
	if svc.ExpiryTimeUnixMs() <= 0 {
		t.Fatalf("expected positive expiry unix ms")
	}

	expiredSvc := domain.Service{
		ID:          2,
		ClientEmail: "user_expired",
		ExpiryTime:  now.Add(-1 * time.Hour),
		Status:      domain.ServiceStatusActive,
	}
	if !expiredSvc.IsExpired() {
		t.Fatalf("service should be expired")
	}
	if expiredSvc.IsActive() {
		t.Fatalf("expired service should not be active")
	}

	noExpirySvc := domain.Service{
		ID:          3,
		ClientEmail: "user_no_expiry",
		ExpiryTime:  time.Time{},
		Status:      domain.ServiceStatusActive,
	}
	if noExpirySvc.IsExpired() {
		t.Fatalf("service with zero expiry should not be expired")
	}
	if noExpirySvc.ExpiryTimeUnixMs() != 0 {
		t.Fatalf("zero expiry should return 0 ms")
	}
}

func TestUserDomain(t *testing.T) {
	u1 := domain.User{
		Username:  "johndoe",
		FirstName: "John",
		LastName:  "Doe",
	}
	if u1.FullName() != "John Doe" {
		t.Fatalf("expected 'John Doe', got %q", u1.FullName())
	}

	u2 := domain.User{
		Username:  "alice",
		FirstName: "Alice",
	}
	if u2.FullName() != "Alice" {
		t.Fatalf("expected 'Alice', got %q", u2.FullName())
	}

	u3 := domain.User{
		Username: "onlyuser",
	}
	if u3.FullName() != "@onlyuser" {
		t.Fatalf("expected '@onlyuser', got %q", u3.FullName())
	}
}

func TestDomainJSONSerialization(t *testing.T) {
	order := domain.Order{
		ID:             10,
		InstanceID:     1,
		UserID:         42,
		TelegramID:     999999,
		Type:           domain.OrderTypePurchase,
		DurationMonths: 1,
		UserCount:      2,
		Amount:         money.NewToman(200_000),
		PaymentMethod:  domain.PaymentMethodWallet,
		Status:         domain.OrderStatusApproved,
	}

	bytes, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("failed to marshal order: %v", err)
	}

	var restored domain.Order
	if err := json.Unmarshal(bytes, &restored); err != nil {
		t.Fatalf("failed to unmarshal order: %v", err)
	}

	if restored.ID != order.ID || !restored.Amount.Equal(order.Amount) {
		t.Fatalf("restored order mismatch: %+v vs %+v", restored, order)
	}
}
