package usecase_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"qomod/services/users/internal/domain"
	"qomod/services/users/internal/usecase"
)

// --- Mock Implementations ---

type mockUserRepo struct {
	users   map[uuid.UUID]*domain.User
	byPhone map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:   make(map[uuid.UUID]*domain.User),
		byPhone: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	if _, exists := m.byPhone[u.CellphoneNumber]; exists {
		return domain.ErrPhoneAlreadyExists
	}
	m.users[u.ID] = u
	m.byPhone[u.CellphoneNumber] = u
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByCellphone(ctx context.Context, cellphone string) (*domain.User, error) {
	u, ok := m.byPhone[cellphone]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email != nil && *u.Email == email {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) UpdateProfile(ctx context.Context, u *domain.User) error {
	existing, ok := m.users[u.ID]
	if !ok {
		return domain.ErrUserNotFound
	}
	existing.FirstName = u.FirstName
	existing.LastName = u.LastName
	existing.Birthdate = u.Birthdate
	existing.Email = u.Email
	existing.UpdatedAt = u.UpdatedAt
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	existing, ok := m.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	existing.PasswordHash = &passwordHash
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.Status) error {
	existing, ok := m.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	existing.Status = status
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	existing, ok := m.users[id]
	if !ok {
		return domain.ErrUserNotFound
	}
	now := time.Now().UTC()
	existing.DeletedAt = &now
	existing.Status = domain.StatusDeleted
	return nil
}

func (m *mockUserRepo) List(ctx context.Context, filter usecase.ListFilter) ([]*domain.User, int64, error) {
	var list []*domain.User
	for _, u := range m.users {
		if u.DeletedAt != nil {
			continue
		}
		if filter.Status != nil && u.Status != *filter.Status {
			continue
		}
		list = append(list, u)
	}
	return list, int64(len(list)), nil
}

type recordedEvent struct {
	RoutingKey string
	Envelope   domain.EventEnvelope
}

type mockPublisher struct {
	events []recordedEvent
}

func (p *mockPublisher) Publish(ctx context.Context, routingKey string, event domain.EventEnvelope) error {
	p.events = append(p.events, recordedEvent{
		RoutingKey: routingKey,
		Envelope:   event,
	})
	return nil
}

// --- Tests ---

func setupTestUseCase() (*usecase.UserUseCase, *mockUserRepo, *mockPublisher) {
	repo := newMockUserRepo()
	pub := &mockPublisher{}
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := usecase.NewUserUseCase(repo, pub, discardLogger)
	return uc, repo, pub
}

func TestUserUseCase_RegisterUser_Success(t *testing.T) {
	uc, _, pub := setupTestUseCase()
	ctx := context.Background()

	first := "Alice"
	pass := "secretPassword123"

	cmd := usecase.RegisterUserCommand{
		CellphoneNumber: "+12025550199",
		FirstName:       &first,
		Password:        &pass,
	}

	user, err := uc.RegisterUser(ctx, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.CellphoneNumber != cmd.CellphoneNumber {
		t.Errorf("expected phone %s, got %s", cmd.CellphoneNumber, user.CellphoneNumber)
	}
	if user.Status != domain.StatusEnabled {
		t.Errorf("expected status enabled, got %s", user.Status)
	}
	if user.PasswordHash == nil {
		t.Fatal("expected password hash to be populated")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(pass)); err != nil {
		t.Errorf("password verification failed: %v", err)
	}

	// Verify domain event publication
	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event published, got %d", len(pub.events))
	}
	if pub.events[0].RoutingKey != domain.EventUserRegistered {
		t.Errorf("expected routing key %s, got %s", domain.EventUserRegistered, pub.events[0].RoutingKey)
	}
	if pub.events[0].Envelope.AggregateID != user.ID.String() {
		t.Errorf("expected aggregate id %s, got %s", user.ID.String(), pub.events[0].Envelope.AggregateID)
	}
}

func TestUserUseCase_RegisterUser_InvalidPhone(t *testing.T) {
	uc, _, _ := setupTestUseCase()
	ctx := context.Background()

	cmd := usecase.RegisterUserCommand{
		CellphoneNumber: "invalid-phone",
	}

	_, err := uc.RegisterUser(ctx, cmd)
	if !errors.Is(err, domain.ErrInvalidPhoneNumber) {
		t.Errorf("expected ErrInvalidPhoneNumber, got %v", err)
	}
}

func TestUserUseCase_RegisterUser_DuplicatePhone(t *testing.T) {
	uc, _, _ := setupTestUseCase()
	ctx := context.Background()

	cmd := usecase.RegisterUserCommand{
		CellphoneNumber: "+12025550199",
	}

	_, err := uc.RegisterUser(ctx, cmd)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	_, err = uc.RegisterUser(ctx, cmd)
	if !errors.Is(err, domain.ErrPhoneAlreadyExists) {
		t.Errorf("expected ErrPhoneAlreadyExists, got %v", err)
	}
}

func TestUserUseCase_ChangePassword(t *testing.T) {
	uc, _, pub := setupTestUseCase()
	ctx := context.Background()

	initialPass := "originalPassword1"
	user, err := uc.RegisterUser(ctx, usecase.RegisterUserCommand{
		CellphoneNumber: "+12025550188",
		Password:        &initialPass,
	})
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	// Clear registration event for clarity
	pub.events = nil

	// 1. Attempt change with incorrect old password
	wrongOld := "wrongPassword"
	newPass := "brandNewSecret123"
	err = uc.ChangePassword(ctx, usecase.ChangePasswordCommand{
		ID:          user.ID,
		OldPassword: &wrongOld,
		NewPassword: newPass,
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials on wrong old password, got %v", err)
	}

	// 2. Attempt change with weak new password
	shortPass := "short"
	err = uc.ChangePassword(ctx, usecase.ChangePasswordCommand{
		ID:          user.ID,
		OldPassword: &initialPass,
		NewPassword: shortPass,
	})
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Errorf("expected ErrWeakPassword, got %v", err)
	}

	// 3. Successful password change
	err = uc.ChangePassword(ctx, usecase.ChangePasswordCommand{
		ID:          user.ID,
		OldPassword: &initialPass,
		NewPassword: newPass,
	})
	if err != nil {
		t.Fatalf("unexpected error on valid password change: %v", err)
	}

	// Verify password changed event published
	if len(pub.events) != 1 {
		t.Fatalf("expected 1 password changed event, got %d", len(pub.events))
	}
	if pub.events[0].RoutingKey != domain.EventUserPasswordChanged {
		t.Errorf("expected routing key %s, got %s", domain.EventUserPasswordChanged, pub.events[0].RoutingKey)
	}
}

func TestUserUseCase_UpdateStatus(t *testing.T) {
	uc, _, pub := setupTestUseCase()
	ctx := context.Background()

	user, err := uc.RegisterUser(ctx, usecase.RegisterUserCommand{
		CellphoneNumber: "+12025550177",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	pub.events = nil

	// Update to suspended
	updated, err := uc.UpdateStatus(ctx, usecase.UpdateStatusCommand{
		ID:     user.ID,
		Status: domain.StatusSuspended,
	})
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}
	if updated.Status != domain.StatusSuspended {
		t.Errorf("expected status suspended, got %s", updated.Status)
	}

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 status change event, got %d", len(pub.events))
	}
	if pub.events[0].RoutingKey != domain.EventUserStatusChanged {
		t.Errorf("expected routing key %s, got %s", domain.EventUserStatusChanged, pub.events[0].RoutingKey)
	}
}

func TestUserUseCase_DeleteUser(t *testing.T) {
	uc, _, pub := setupTestUseCase()
	ctx := context.Background()

	user, err := uc.RegisterUser(ctx, usecase.RegisterUserCommand{
		CellphoneNumber: "+12025550166",
	})
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}
	pub.events = nil

	// Soft delete
	err = uc.DeleteUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Deleted user should return ErrUserDeleted on GetUserByID
	_, err = uc.GetUserByID(ctx, user.ID)
	if !errors.Is(err, domain.ErrUserDeleted) {
		t.Errorf("expected ErrUserDeleted after soft delete, got %v", err)
	}

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 delete event, got %d", len(pub.events))
	}
	if pub.events[0].RoutingKey != domain.EventUserDeleted {
		t.Errorf("expected routing key %s, got %s", domain.EventUserDeleted, pub.events[0].RoutingKey)
	}
}
