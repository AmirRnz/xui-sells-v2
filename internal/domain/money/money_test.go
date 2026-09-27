package money_test

import (
	"encoding/json"
	"testing"

	"xui-sells-v2/internal/domain/money"
)

func TestTomanInputScale(t *testing.T) {
	// An input of 200 represents 200,000 Toman.
	m := money.FromTomanInput(200)
	if m.Amount != 200_000 {
		t.Fatalf("expected amount 200000, got %d", m.Amount)
	}
	if m.Currency != money.CurrencyToman {
		t.Fatalf("expected currency TOMAN, got %s", m.Currency)
	}

	// Converting back to input scale
	input, err := m.ToTomanInput()
	if err != nil {
		t.Fatalf("unexpected error converting to input scale: %v", err)
	}
	if input != 200 {
		t.Fatalf("expected input 200, got %d", input)
	}

	// Buying five such units costs 1 million toman (5 * 200 = 1000 input units = 1,000,000 Toman)
	total := m.Mul(5)
	if total.Amount != 1_000_000 {
		t.Fatalf("expected 1000000, got %d", total.Amount)
	}

	// Must format as "1 میلیون تومان" in Persian, "1 million toman" in English
	fa := total.Format("fa")
	if fa != "1 میلیون تومان" {
		t.Fatalf("expected '1 میلیون تومان', got '%s'", fa)
	}
	if fa == "1000 هزار تومان" {
		t.Fatalf("must never format as '1000 هزار تومان'")
	}

	en := total.Format("en")
	if en != "1 million toman" {
		t.Fatalf("expected '1 million toman', got '%s'", en)
	}
}

func TestTomanFormattingPersian(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		expected string
	}{
		{"200 Thousand", 200_000, "200 هزار تومان"},
		{"1 Million", 1_000_000, "1 میلیون تومان"},
		{"5 Million", 5_000_000, "5 میلیون تومان"},
		{"1 Billion", 1_000_000_000, "1 میلیارد تومان"},
		{"Zero", 0, "0 تومان"},
		{"Under 1000", 500, "500 تومان"},
		{"1 Thousand", 1_000, "1 هزار تومان"},
		{"1 Million 200 Thousand", 1_200_000, "1 میلیون و 200 هزار تومان"},
		{"Negative 200 Thousand", -200_000, "-200 هزار تومان"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := money.FormatToman(tt.amount, "fa")
			if got != tt.expected {
				t.Errorf("FormatToman(%d, fa) = %q, want %q", tt.amount, got, tt.expected)
			}
			if tt.amount == 1_000_000 && got == "1000 هزار تومان" {
				t.Errorf("1,000,000 must never format as '1000 هزار تومان'")
			}
		})
	}
}

func TestTomanFormattingEnglish(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		expected string
	}{
		{"200 Thousand", 200_000, "200 thousand toman"},
		{"1 Million", 1_000_000, "1 million toman"},
		{"5 Million", 5_000_000, "5 million toman"},
		{"1 Billion", 1_000_000_000, "1 billion toman"},
		{"Zero", 0, "0 toman"},
		{"Under 1000", 500, "500 toman"},
		{"1 Thousand", 1_000, "1 thousand toman"},
		{"1 Million 200 Thousand", 1_200_000, "1 million and 200 thousand toman"},
		{"Negative 200 Thousand", -200_000, "-200 thousand toman"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := money.FormatToman(tt.amount, "en")
			if got != tt.expected {
				t.Errorf("FormatToman(%d, en) = %q, want %q", tt.amount, got, tt.expected)
			}
		})
	}
}

func TestUSDFormatting(t *testing.T) {
	tests := []struct {
		name     string
		cents    int64
		expected string
	}{
		{"1 Dollar", 100, "$1.00"},
		{"2 Dollars 50 Cents", 250, "$2.50"},
		{"5 Cents", 5, "$0.05"},
		{"Zero Cents", 0, "$0.00"},
		{"Negative 1 Dollar", -100, "-$1.00"},
		{"Negative 50 Cents", -50, "-$0.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := money.NewUSD(tt.cents)
			got := m.Format("en")
			if got != tt.expected {
				t.Errorf("USD Format(%d) = %q, want %q", tt.cents, got, tt.expected)
			}
			gotFa := m.Format("fa")
			if gotFa != tt.expected {
				t.Errorf("USD Format FA(%d) = %q, want %q", tt.cents, gotFa, tt.expected)
			}
		})
	}
}

func TestMoneyArithmetic(t *testing.T) {
	m1 := money.NewToman(200_000)
	m2 := money.NewToman(300_000)

	// Add
	sum, err := m1.Add(m2)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if sum.Amount != 500_000 {
		t.Fatalf("expected sum 500000, got %d", sum.Amount)
	}

	// Sub
	diff, err := m2.Sub(m1)
	if err != nil {
		t.Fatalf("Sub failed: %v", err)
	}
	if diff.Amount != 100_000 {
		t.Fatalf("expected diff 100000, got %d", diff.Amount)
	}

	// Mul
	mul := m1.Mul(3)
	if mul.Amount != 600_000 {
		t.Fatalf("expected mul 600000, got %d", mul.Amount)
	}

	// Div
	div, err := m1.Div(2)
	if err != nil {
		t.Fatalf("Div failed: %v", err)
	}
	if div.Amount != 100_000 {
		t.Fatalf("expected div 100000, got %d", div.Amount)
	}

	// Div by zero
	_, err = m1.Div(0)
	if err != money.ErrDivisionByZero {
		t.Fatalf("expected ErrDivisionByZero, got %v", err)
	}

	// Mismatched currency
	usd := money.NewUSD(100)
	_, err = m1.Add(usd)
	if err != money.ErrCurrencyMismatch {
		t.Fatalf("expected ErrCurrencyMismatch on Add, got %v", err)
	}
	_, err = m1.Sub(usd)
	if err != money.ErrCurrencyMismatch {
		t.Fatalf("expected ErrCurrencyMismatch on Sub, got %v", err)
	}
}

func TestMoneyComparisons(t *testing.T) {
	m1 := money.NewToman(200_000)
	m2 := money.NewToman(300_000)
	m3 := money.NewToman(200_000)
	zero := money.NewToman(0)
	neg := money.NewToman(-50)

	if !m1.Equal(m3) {
		t.Fatalf("expected m1 equal to m3")
	}
	if m1.Equal(m2) {
		t.Fatalf("m1 should not equal m2")
	}
	if !m2.GreaterThan(m1) {
		t.Fatalf("expected m2 > m1")
	}
	if !m1.LessThan(m2) {
		t.Fatalf("expected m1 < m2")
	}
	if !zero.IsZero() {
		t.Fatalf("expected zero.IsZero() == true")
	}
	if !m1.IsPositive() {
		t.Fatalf("expected m1.IsPositive() == true")
	}
	if !neg.IsNegative() {
		t.Fatalf("expected neg.IsNegative() == true")
	}
}

func TestMoneyJSONSerialization(t *testing.T) {
	m := money.NewToman(200_000)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var restored money.Money
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !m.Equal(restored) {
		t.Fatalf("restored money %v does not equal original %v", restored, m)
	}
}
