package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Status represents the operational state of a user account.
type Status string

const (
	StatusEnabled   Status = "enabled"
	StatusDisabled  Status = "disabled"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

// IsValid checks whether the status belongs to supported enum values.
func (s Status) IsValid() bool {
	switch s {
	case StatusEnabled, StatusDisabled, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}

// User is the core aggregate entity representing a system user.
type User struct {
	ID              uuid.UUID  `json:"id"`
	CellphoneNumber string     `json:"cellphone_number"`
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	Birthdate       *time.Time `json:"birthdate,omitempty"`
	Email           *string    `json:"email,omitempty"`
	PasswordHash    *string    `json:"-"` // Never expose password hash in JSON serialization
	Status          Status     `json:"status"`
	PhoneVerified   bool       `json:"phone_verified"`
	EmailVerified   bool       `json:"email_verified"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// FullName returns the concatenated first and last name if available.
func (u *User) FullName() string {
	var parts []string
	if u.FirstName != nil && *u.FirstName != "" {
		parts = append(parts, *u.FirstName)
	}
	if u.LastName != nil && *u.LastName != "" {
		parts = append(parts, *u.LastName)
	}
	return strings.Join(parts, " ")
}

// IsActive indicates whether the user account is active and able to authenticate.
func (u *User) IsActive() bool {
	return u.Status == StatusEnabled && u.DeletedAt == nil
}

// HasPassword checks whether a password hash has been established for this user.
func (u *User) HasPassword() bool {
	return u.PasswordHash != nil && *u.PasswordHash != ""
}
