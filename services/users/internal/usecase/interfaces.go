package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"qomod/services/users/internal/domain"
)

// ListFilter parameters for paginated user queries.
type ListFilter struct {
	Limit  int
	Offset int
	Status *domain.Status
}

// UserRepository specifies data store interactions for User aggregates.
// Idiomatic Go: Defined at the usecase boundary where business logic consumes it.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByCellphone(ctx context.Context, cellphone string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateProfile(ctx context.Context, user *domain.User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter ListFilter) ([]*domain.User, int64, error)
}

// EventPublisher abstracts asynchronous event publication (e.g., to RabbitMQ).
type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, event domain.EventEnvelope) error
}

// Input commands for UseCase methods

type RegisterUserCommand struct {
	CellphoneNumber string     `json:"cellphone_number"`
	FirstName       *string    `json:"first_name"`
	LastName        *string    `json:"last_name"`
	Birthdate       *time.Time `json:"birthdate"`
	Email           *string    `json:"email"`
	Password        *string    `json:"password"`
}

type UpdateProfileCommand struct {
	ID        uuid.UUID  `json:"id"`
	FirstName *string    `json:"first_name"`
	LastName  *string    `json:"last_name"`
	Birthdate *time.Time `json:"birthdate"`
	Email     *string    `json:"email"`
}

type ChangePasswordCommand struct {
	ID          uuid.UUID `json:"id"`
	OldPassword *string   `json:"old_password"` // Optional if user had no prior password
	NewPassword string    `json:"new_password"`
}

type UpdateStatusCommand struct {
	ID     uuid.UUID     `json:"id"`
	Status domain.Status `json:"status"`
}
