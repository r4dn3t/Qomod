package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"qomod/services/users/internal/domain"
)

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   domain.Status
		expected bool
	}{
		{"valid enabled", domain.StatusEnabled, true},
		{"valid disabled", domain.StatusDisabled, true},
		{"valid suspended", domain.StatusSuspended, true},
		{"valid deleted", domain.StatusDeleted, true},
		{"invalid empty", domain.Status(""), false},
		{"invalid random", domain.Status("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expected {
				t.Errorf("Status.IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_FullName(t *testing.T) {
	first := "John"
	last := "Doe"

	tests := []struct {
		name      string
		firstName *string
		lastName  *string
		expected  string
	}{
		{"both present", &first, &last, "John Doe"},
		{"only first", &first, nil, "John"},
		{"only last", nil, &last, "Doe"},
		{"both nil", nil, nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &domain.User{
				FirstName: tt.firstName,
				LastName:  tt.lastName,
			}
			if got := u.FullName(); got != tt.expected {
				t.Errorf("User.FullName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		status    domain.Status
		deletedAt *time.Time
		expected  bool
	}{
		{"active user", domain.StatusEnabled, nil, true},
		{"disabled user", domain.StatusDisabled, nil, false},
		{"suspended user", domain.StatusSuspended, nil, false},
		{"soft deleted user", domain.StatusEnabled, &now, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &domain.User{
				ID:        uuid.New(),
				Status:    tt.status,
				DeletedAt: tt.deletedAt,
			}
			if got := u.IsActive(); got != tt.expected {
				t.Errorf("User.IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}
