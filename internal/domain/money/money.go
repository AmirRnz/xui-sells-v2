package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Currency represents a supported currency.
type Currency string

const (
	CurrencyToman Currency = "TOMAN"
	CurrencyIRT   Currency = "IRT" // alias for TOMAN
	CurrencyUSD   Currency = "USD"
)

// TomanInputScale defines the scale multiplier for Toman input.
// An input of 200 represents 200,000 Toman.
const TomanInputScale int64 = 1000

var (
	ErrCurrencyMismatch = errors.New("currency mismatch")
	ErrDivisionByZero   = errors.New("division by zero")
	ErrInvalidCurrency  = errors.New("invalid or unsupported currency")
)

// Money represents an exact integer monetary amount in a specific currency.
// For Toman, Amount is in exact full Toman units (e.g. 200,000).
// For USD, Amount is in exact cents (e.g. 100 for $1.00).
type Money struct {
	Amount   int64    `json:"amount"`
	Currency Currency `json:"currency"`
}

// New creates a new Money value object.
func New(amount int64, currency Currency) Money {
	return Money{
		Amount:   amount,
		Currency: NormalizeCurrency(currency),
	}
}

// NewToman creates a Money instance in Toman with the given full amount.
func NewToman(amount int64) Money {
	return Money{
		Amount:   amount,
		Currency: CurrencyToman,
	}
}

// FromTomanInput converts a shorthand user/admin input into Toman Money.
// For example, input 200 becomes 200,000 Toman.
func FromTomanInput(input int64) Money {
	return NewToman(input * TomanInputScale)
}

// ToTomanInput converts Toman Money back to its input scale representation.
// For example, 200,000 Toman returns 200.
func (m Money) ToTomanInput() (int64, error) {
	if NormalizeCurrency(m.Currency) != CurrencyToman {
		return 0, ErrInvalidCurrency
	}
	return m.Amount / TomanInputScale, nil
}

// NewUSD creates a Money instance in USD with the given amount in cents.
func NewUSD(cents int64) Money {
	return Money{
		Amount:   cents,
		Currency: CurrencyUSD,
	}
}

// FromUSDCents is an alias for NewUSD.
func FromUSDCents(cents int64) Money {
	return NewUSD(cents)
}

// FromUSD creates a Money instance in USD from whole dollars.
func FromUSD(dollars int64) Money {
	return NewUSD(dollars * 100)
}

// NormalizeCurrency standardizes currency strings.
func NormalizeCurrency(c Currency) Currency {
	upper := strings.ToUpper(strings.TrimSpace(string(c)))
	switch upper {
	case "TOMAN", "IRT":
		return CurrencyToman
	case "USD":
		return CurrencyUSD
	default:
		return Currency(upper)
	}
}

// Add adds two Money values of the same currency.
func (m Money) Add(other Money) (Money, error) {
	if NormalizeCurrency(m.Currency) != NormalizeCurrency(other.Currency) {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: NormalizeCurrency(m.Currency),
	}, nil
}

// Sub subtracts other Money from m.
func (m Money) Sub(other Money) (Money, error) {
	if NormalizeCurrency(m.Currency) != NormalizeCurrency(other.Currency) {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{
		Amount:   m.Amount - other.Amount,
		Currency: NormalizeCurrency(m.Currency),
	}, nil
}

// Mul multiplies Money by an integer factor.
func (m Money) Mul(factor int64) Money {
	return Money{
		Amount:   m.Amount * factor,
		Currency: NormalizeCurrency(m.Currency),
	}
}

// Div divides Money by an integer divisor.
func (m Money) Div(divisor int64) (Money, error) {
	if divisor == 0 {
		return Money{}, ErrDivisionByZero
	}
	return Money{
		Amount:   m.Amount / divisor,
		Currency: NormalizeCurrency(m.Currency),
	}, nil
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsPositive returns true if amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.Amount > 0
}

// IsNegative returns true if amount is less than zero.
func (m Money) IsNegative() bool {
	return m.Amount < 0
}

// Equal returns true if amounts and currencies are identical.
func (m Money) Equal(other Money) bool {
	return m.Amount == other.Amount && NormalizeCurrency(m.Currency) == NormalizeCurrency(other.Currency)
}

// GreaterThan returns true if m is strictly greater than other.
func (m Money) GreaterThan(other Money) bool {
	if NormalizeCurrency(m.Currency) != NormalizeCurrency(other.Currency) {
		return false
	}
	return m.Amount > other.Amount
}

// LessThan returns true if m is strictly less than other.
func (m Money) LessThan(other Money) bool {
	if NormalizeCurrency(m.Currency) != NormalizeCurrency(other.Currency) {
		return false
	}
	return m.Amount < other.Amount
}

// Format formats the money amount according to the given language ("fa" or "en").
func (m Money) Format(lang string) string {
	return FormatCurrency(m, lang)
}

// String implements fmt.Stringer using English formatting.
func (m Money) String() string {
	return m.Format("en")
}

// FormatCurrency formats a Money instance according to currency and language.
func FormatCurrency(m Money, lang string) string {
	switch NormalizeCurrency(m.Currency) {
	case CurrencyUSD:
		return FormatUSD(m.Amount)
	case CurrencyToman:
		return FormatToman(m.Amount, lang)
	default:
		return fmt.Sprintf("%d %s", m.Amount, m.Currency)
	}
}

// FormatUSD formats a cent amount as "$1.00", "$0.50", "-$1.00", etc.
func FormatUSD(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	dollars := cents / 100
	remainder := cents % 100
	return fmt.Sprintf("%s$%d.%02d", sign, dollars, remainder)
}

// FormatToman formats a full Toman amount into human-readable text.
// Persian examples: 200,000 -> "200 هزار تومان"; 1,000,000 -> "1 میلیون تومان". Never "1000 هزار تومان".
// English examples: 200,000 -> "200 thousand toman"; 1,000,000 -> "1 million toman".
func FormatToman(amount int64, lang string) string {
	isPersian := strings.HasPrefix(strings.ToLower(strings.TrimSpace(lang)), "fa")

	if amount == 0 {
		if isPersian {
			return "0 تومان"
		}
		return "0 toman"
	}

	prefix := ""
	abs := amount
	if abs < 0 {
		prefix = "-"
		abs = -abs
	}

	billions := abs / 1_000_000_000
	millions := (abs % 1_000_000_000) / 1_000_000
	thousands := (abs % 1_000_000) / 1_000
	units := abs % 1_000

	if isPersian {
		var parts []string
		if billions > 0 {
			parts = append(parts, fmt.Sprintf("%d میلیارد", billions))
		}
		if millions > 0 {
			parts = append(parts, fmt.Sprintf("%d میلیون", millions))
		}
		if thousands > 0 {
			parts = append(parts, fmt.Sprintf("%d هزار", thousands))
		}
		if units > 0 {
			parts = append(parts, fmt.Sprintf("%d", units))
		}

		if len(parts) == 0 {
			return "0 تومان"
		}
		return prefix + strings.Join(parts, " و ") + " تومان"
	}

	var parts []string
	if billions > 0 {
		parts = append(parts, fmt.Sprintf("%d billion", billions))
	}
	if millions > 0 {
		parts = append(parts, fmt.Sprintf("%d million", millions))
	}
	if thousands > 0 {
		parts = append(parts, fmt.Sprintf("%d thousand", thousands))
	}
	if units > 0 {
		parts = append(parts, fmt.Sprintf("%d", units))
	}

	if len(parts) == 0 {
		return "0 toman"
	}
	return prefix + strings.Join(parts, " and ") + " toman"
}

// MarshalJSON custom JSON marshaling.
func (m Money) MarshalJSON() ([]byte, error) {
	type Alias Money
	return json.Marshal(&struct {
		Currency string `json:"currency"`
		Alias
	}{
		Currency: string(NormalizeCurrency(m.Currency)),
		Alias:    (Alias)(m),
	})
}
