package domain

import (
	"time"

	"github.com/google/uuid"
)

// Event type constants used for routing keys in RabbitMQ.
const (
	EventUserRegistered     = "user.registered"
	EventUserProfileUpdated = "user.profile.updated"
	EventUserPasswordChanged = "user.password.changed"
	EventUserStatusChanged  = "user.status.changed"
	EventUserDeleted        = "user.deleted"
)

// EventEnvelope provides a standardized CloudEvents-inspired wrapper for all published messages.
type EventEnvelope struct {
	EventID       string    `json:"event_id"`
	EventType     string    `json:"event_type"`
	AggregateType string    `json:"aggregate_type"` // e.g. "user"
	AggregateID   string    `json:"aggregate_id"`
	Timestamp     time.Time `json:"timestamp"`
	Version       string    `json:"version"` // Schema version, e.g. "1.0"
	Data          any       `json:"data"`
}

// NewEventEnvelope creates a fresh event envelope with a unique ID and current timestamp.
func NewEventEnvelope(eventType, aggregateID string, data any) EventEnvelope {
	return EventEnvelope{
		EventID:       uuid.New().String(),
		EventType:     eventType,
		AggregateType: "user",
		AggregateID:   aggregateID,
		Timestamp:     time.Now().UTC(),
		Version:       "1.0",
		Data:          data,
	}
}

// UserRegisteredData represents the payload for user.registered event.
type UserRegisteredData struct {
	UserID          string     `json:"user_id"`
	CellphoneNumber string     `json:"cellphone_number"`
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	Birthdate       *time.Time `json:"birthdate,omitempty"`
	Email           *string    `json:"email,omitempty"`
	Status          Status     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
}

// UserProfileUpdatedData represents the payload for user.profile.updated event.
type UserProfileUpdatedData struct {
	UserID          string     `json:"user_id"`
	CellphoneNumber string     `json:"cellphone_number"`
	FirstName       *string    `json:"first_name,omitempty"`
	LastName        *string    `json:"last_name,omitempty"`
	Birthdate       *time.Time `json:"birthdate,omitempty"`
	Email           *string    `json:"email,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// UserPasswordChangedData represents the payload for user.password.changed event.
type UserPasswordChangedData struct {
	UserID    string    `json:"user_id"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserStatusChangedData represents the payload for user.status.changed event.
type UserStatusChangedData struct {
	UserID    string    `json:"user_id"`
	OldStatus Status    `json:"old_status"`
	NewStatus Status    `json:"new_status"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserDeletedData represents the payload for user.deleted event.
type UserDeletedData struct {
	UserID    string    `json:"user_id"`
	DeletedAt time.Time `json:"deleted_at"`
}
