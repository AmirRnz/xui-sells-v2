package domain

import (
	"time"

	"xui-sells-v2/internal/domain/money"
)

// Language represents the supported bot languages.
type Language string

const (
	LangFA Language = "fa"
	LangEN Language = "en"
)

// User represents a Telegram user inside an instance.
type User struct {
	ID                 int64       `json:"id"`
	InstanceID         int64       `json:"instance_id"`
	TelegramID         int64       `json:"telegram_id"`
	Username           string      `json:"username,omitempty"`
	FirstName          string      `json:"first_name,omitempty"`
	LastName           string      `json:"last_name,omitempty"`
	Language           Language    `json:"language"`
	WalletBalance      money.Money `json:"wallet_balance"`
	ReferrerTelegramID *int64      `json:"referrer_telegram_id,omitempty"`
	IsAdmin            bool        `json:"is_admin"`
	IsBlocked          bool        `json:"is_blocked"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// FullName returns the combined first and last name.
func (u User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		if u.Username != "" {
			return "@" + u.Username
		}
		return ""
	}
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}
