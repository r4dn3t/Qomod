package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"qomod/services/users/internal/domain"
)

// Regex for international E.164 phone numbers (e.g., +1234567890 to +12345678901234)
var e164Regex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

// Basic email validation regex
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// UserUseCase implements application logic for user operations.
type UserUseCase struct {
	repo      UserRepository
	publisher EventPublisher
	logger    *slog.Logger
}

// NewUserUseCase constructor injecting dependencies.
func NewUserUseCase(repo UserRepository, publisher EventPublisher, logger *slog.Logger) *UserUseCase {
	return &UserUseCase{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

// RegisterUser handles creation of a new user account, valid cell number, password hashing, and event publishing.
func (uc *UserUseCase) RegisterUser(ctx context.Context, cmd RegisterUserCommand) (*domain.User, error) {
	// 1. Sanitize & Validate Phone Number
	phone := strings.TrimSpace(cmd.CellphoneNumber)
	if !e164Regex.MatchString(phone) {
		return nil, domain.ErrInvalidPhoneNumber
	}

	// 2. Validate Email if provided
	var email *string
	if cmd.Email != nil && strings.TrimSpace(*cmd.Email) != "" {
		cleanedEmail := strings.ToLower(strings.TrimSpace(*cmd.Email))
		if !emailRegex.MatchString(cleanedEmail) {
			return nil, fmt.Errorf("invalid email format: %w", domain.ErrInvalidCredentials)
		}
		email = &cleanedEmail
	}

	// 3. Hash Password if provided
	var passwordHash *string
	if cmd.Password != nil && *cmd.Password != "" {
		if len(*cmd.Password) < 8 {
			return nil, domain.ErrWeakPassword
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*cmd.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		hashStr := string(hash)
		passwordHash = &hashStr
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:              uuid.New(),
		CellphoneNumber: phone,
		FirstName:       cmd.FirstName,
		LastName:        cmd.LastName,
		Birthdate:       cmd.Birthdate,
		Email:           email,
		PasswordHash:    passwordHash,
		Status:          domain.StatusEnabled,
		PhoneVerified:   false,
		EmailVerified:   false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 4. Persist User in Repository
	if err := uc.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// 5. Emit user.registered Domain Event
	eventData := domain.UserRegisteredData{
		UserID:          user.ID.String(),
		CellphoneNumber: user.CellphoneNumber,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		Birthdate:       user.Birthdate,
		Email:           user.Email,
		Status:          user.Status,
		CreatedAt:       user.CreatedAt,
	}
	envelope := domain.NewEventEnvelope(domain.EventUserRegistered, user.ID.String(), eventData)

	if uc.publisher != nil {
		if err := uc.publisher.Publish(ctx, domain.EventUserRegistered, envelope); err != nil {
			uc.logger.Error("failed to publish user.registered event", "user_id", user.ID, "error", err)
			// Non-blocking in basic flow: user is registered, but event publication error is logged
		}
	}

	return user, nil
}

// GetUserByID retrieves a user record by their unique ID.
func (uc *UserUseCase) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user.DeletedAt != nil {
		return nil, domain.ErrUserDeleted
	}
	return user, nil
}

// UpdateProfile updates non-credential personal attributes and emits user.profile.updated event.
func (uc *UserUseCase) UpdateProfile(ctx context.Context, cmd UpdateProfileCommand) (*domain.User, error) {
	existing, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if existing.DeletedAt != nil {
		return nil, domain.ErrUserDeleted
	}

	var email *string
	if cmd.Email != nil && strings.TrimSpace(*cmd.Email) != "" {
		cleaned := strings.ToLower(strings.TrimSpace(*cmd.Email))
		if !emailRegex.MatchString(cleaned) {
			return nil, fmt.Errorf("invalid email format: %w", domain.ErrInvalidCredentials)
		}
		email = &cleaned
	}

	existing.FirstName = cmd.FirstName
	existing.LastName = cmd.LastName
	existing.Birthdate = cmd.Birthdate
	existing.Email = email
	existing.UpdatedAt = time.Now().UTC()

	if err := uc.repo.UpdateProfile(ctx, existing); err != nil {
		return nil, err
	}

	eventData := domain.UserProfileUpdatedData{
		UserID:          existing.ID.String(),
		CellphoneNumber: existing.CellphoneNumber,
		FirstName:       existing.FirstName,
		LastName:        existing.LastName,
		Birthdate:       existing.Birthdate,
		Email:           existing.Email,
		UpdatedAt:       existing.UpdatedAt,
	}
	envelope := domain.NewEventEnvelope(domain.EventUserProfileUpdated, existing.ID.String(), eventData)

	if uc.publisher != nil {
		if err := uc.publisher.Publish(ctx, domain.EventUserProfileUpdated, envelope); err != nil {
			uc.logger.Error("failed to publish user.profile.updated event", "user_id", existing.ID, "error", err)
		}
	}

	return existing, nil
}

// ChangePassword changes or sets the user's password securely.
func (uc *UserUseCase) ChangePassword(ctx context.Context, cmd ChangePasswordCommand) error {
	if len(cmd.NewPassword) < 8 {
		return domain.ErrWeakPassword
	}

	user, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if user.DeletedAt != nil {
		return domain.ErrUserDeleted
	}

	// If existing user already has a password, verify old password
	if user.HasPassword() {
		if cmd.OldPassword == nil || *cmd.OldPassword == "" {
			return fmt.Errorf("old password required: %w", domain.ErrInvalidCredentials)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(*cmd.OldPassword)); err != nil {
			return fmt.Errorf("incorrect old password: %w", domain.ErrInvalidCredentials)
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cmd.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := uc.repo.UpdatePassword(ctx, cmd.ID, string(hash)); err != nil {
		return err
	}

	eventData := domain.UserPasswordChangedData{
		UserID:    user.ID.String(),
		UpdatedAt: time.Now().UTC(),
	}
	envelope := domain.NewEventEnvelope(domain.EventUserPasswordChanged, user.ID.String(), eventData)

	if uc.publisher != nil {
		if err := uc.publisher.Publish(ctx, domain.EventUserPasswordChanged, envelope); err != nil {
			uc.logger.Error("failed to publish user.password.changed event", "user_id", user.ID, "error", err)
		}
	}

	return nil
}

// UpdateStatus changes account status (enabled, disabled, suspended) and emits event.
func (uc *UserUseCase) UpdateStatus(ctx context.Context, cmd UpdateStatusCommand) (*domain.User, error) {
	if !cmd.Status.IsValid() || cmd.Status == domain.StatusDeleted {
		return nil, domain.ErrInvalidStatus
	}

	user, err := uc.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}
	if user.DeletedAt != nil {
		return nil, domain.ErrUserDeleted
	}

	oldStatus := user.Status
	if oldStatus == cmd.Status {
		return user, nil // No-op if unchanged
	}

	if err := uc.repo.UpdateStatus(ctx, cmd.ID, cmd.Status); err != nil {
		return nil, err
	}

	user.Status = cmd.Status
	user.UpdatedAt = time.Now().UTC()

	eventData := domain.UserStatusChangedData{
		UserID:    user.ID.String(),
		OldStatus: oldStatus,
		NewStatus: cmd.Status,
		UpdatedAt: user.UpdatedAt,
	}
	envelope := domain.NewEventEnvelope(domain.EventUserStatusChanged, user.ID.String(), eventData)

	if uc.publisher != nil {
		if err := uc.publisher.Publish(ctx, domain.EventUserStatusChanged, envelope); err != nil {
			uc.logger.Error("failed to publish user.status.changed event", "user_id", user.ID, "error", err)
		}
	}

	return user, nil
}

// DeleteUser performs a soft delete and emits user.deleted event.
func (uc *UserUseCase) DeleteUser(ctx context.Context, id uuid.UUID) error {
	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user.DeletedAt != nil {
		return domain.ErrUserDeleted
	}

	if err := uc.repo.SoftDelete(ctx, id); err != nil {
		return err
	}

	eventData := domain.UserDeletedData{
		UserID:    id.String(),
		DeletedAt: time.Now().UTC(),
	}
	envelope := domain.NewEventEnvelope(domain.EventUserDeleted, id.String(), eventData)

	if uc.publisher != nil {
		if err := uc.publisher.Publish(ctx, domain.EventUserDeleted, envelope); err != nil {
			uc.logger.Error("failed to publish user.deleted event", "user_id", id, "error", err)
		}
	}

	return nil
}

// ListUsers queries users with pagination and optional status filter.
func (uc *UserUseCase) ListUsers(ctx context.Context, filter ListFilter) ([]*domain.User, int64, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return uc.repo.List(ctx, filter)
}
