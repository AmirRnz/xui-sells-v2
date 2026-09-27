package domain

import (
	"time"
)

// ServiceStatus represents the current status of a VPN subscription.
type ServiceStatus string

const (
	ServiceStatusActive   ServiceStatus = "active"
	ServiceStatusExpired  ServiceStatus = "expired"
	ServiceStatusDisabled ServiceStatus = "disabled"
	ServiceStatusRevoked  ServiceStatus = "revoked"
)

// Service represents a VPN subscription client attached to 3x-ui.
type Service struct {
	ID          int64         `json:"id"`
	InstanceID  int64         `json:"instance_id"`
	UserID      int64         `json:"user_id"`
	TelegramID  int64         `json:"telegram_id"`
	PlanID      int64         `json:"plan_id"`
	ClientEmail string        `json:"client_email"`
	ClientUUID  string        `json:"client_uuid"`
	SubID       string        `json:"sub_id"`
	GroupName   string        `json:"group_name"`
	InboundIDs  []int         `json:"inbound_ids"`
	TotalBytes  int64         `json:"total_bytes"`
	LimitIP     int           `json:"limit_ip"`
	ExpiryTime  time.Time     `json:"expiry_time"`
	Status      ServiceStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// IsExpired returns true if the service has passed its expiration time.
func (s Service) IsExpired() bool {
	if s.ExpiryTime.IsZero() {
		return false // 0 means no expiration
	}
	return time.Now().After(s.ExpiryTime)
}

// IsActive returns true if the service is active and not expired.
func (s Service) IsActive() bool {
	return s.Status == ServiceStatusActive && !s.IsExpired()
}

// ExpiryTimeUnixMs returns the expiration timestamp in milliseconds as required by 3x-ui.
func (s Service) ExpiryTimeUnixMs() int64 {
	if s.ExpiryTime.IsZero() {
		return 0
	}
	return s.ExpiryTime.UnixNano() / int64(time.Millisecond)
}
