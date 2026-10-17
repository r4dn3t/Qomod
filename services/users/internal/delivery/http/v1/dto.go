package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"qomod/services/users/internal/domain"
	"qomod/services/users/internal/usecase"
)

// Date represents a date-only value formatted as YYYY-MM-DD.
type Date time.Time

// UnmarshalJSON parses a JSON string into Date.
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	*d = Date(t)
	return nil
}

// MarshalJSON formats Date as YYYY-MM-DD JSON string.
func (d Date) MarshalJSON() ([]byte, error) {
	t := time.Time(d)
	if t.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", t.Format("2006-01-02"))), nil
}

// Time converts Date to time.Time pointer.
func (d *Date) Time() *time.Time {
	if d == nil {
		return nil
	}
	t := time.Time(*d)
	return &t
}

// CreateUserRequest is the JSON payload for user registration.
type CreateUserRequest struct {
	CellphoneNumber string  `json:"cellphone_number"`
	FirstName       *string `json:"first_name,omitempty"`
	LastName        *string `json:"last_name,omitempty"`
	Birthdate       *Date   `json:"birthdate,omitempty"`
	Email           *string `json:"email,omitempty"`
	Password        *string `json:"password,omitempty"`
}

func (r *CreateUserRequest) ToCommand() usecase.RegisterUserCommand {
	return usecase.RegisterUserCommand{
		CellphoneNumber: r.CellphoneNumber,
		FirstName:       r.FirstName,
		LastName:        r.LastName,
		Birthdate:       r.Birthdate.Time(),
		Email:           r.Email,
		Password:        r.Password,
	}
}

// UpdateProfileRequest is the payload for updating personal information.
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Birthdate *Date   `json:"birthdate,omitempty"`
	Email     *string `json:"email,omitempty"`
}

func (r *UpdateProfileRequest) ToCommand(id uuid.UUID) usecase.UpdateProfileCommand {
	return usecase.UpdateProfileCommand{
		ID:        id,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Birthdate: r.Birthdate.Time(),
		Email:     r.Email,
	}
}

// ChangePasswordRequest payload for password modification.
type ChangePasswordRequest struct {
	OldPassword *string `json:"old_password,omitempty"`
	NewPassword string  `json:"new_password"`
}

// UpdateStatusRequest payload for modifying account status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UserResponse is the sanitized JSON output representing a user.
type UserResponse struct {
	ID              string     `json:"id"`
	CellphoneNumber string     `json:"cellphone_number"`
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	Birthdate       *string    `json:"birthdate,omitempty"`
	Email           *string    `json:"email,omitempty"`
	Status          string     `json:"status"`
	PhoneVerified   bool       `json:"phone_verified"`
	EmailVerified   bool       `json:"email_verified"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ToUserResponse converts domain.User into UserResponse DTO.
func ToUserResponse(u *domain.User) UserResponse {
	var birthdateStr *string
	if u.Birthdate != nil {
		s := u.Birthdate.Format("2006-01-02")
		birthdateStr = &s
	}

	return UserResponse{
		ID:              u.ID.String(),
		CellphoneNumber: u.CellphoneNumber,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		Birthdate:       birthdateStr,
		Email:           u.Email,
		Status:          string(u.Status),
		PhoneVerified:   u.PhoneVerified,
		EmailVerified:   u.EmailVerified,
		LastLoginAt:     u.LastLoginAt,
		CreatedAt:       u.CreatedAt,
		UpdatedAt:       u.UpdatedAt,
	}
}
