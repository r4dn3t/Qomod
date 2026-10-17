package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	delivery "qomod/services/users/internal/delivery/http/httputil"
	v1 "qomod/services/users/internal/delivery/http/v1"
	"qomod/services/users/internal/domain"
	"qomod/services/users/internal/usecase"
)

// In-memory repository mock for HTTP handler tests
type testRepo struct {
	users map[uuid.UUID]*domain.User
}

func newTestRepo() *testRepo {
	return &testRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (r *testRepo) Create(ctx context.Context, u *domain.User) error {
	r.users[u.ID] = u
	return nil
}

func (r *testRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *testRepo) GetByCellphone(ctx context.Context, c string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (r *testRepo) GetByEmail(ctx context.Context, e string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (r *testRepo) UpdateProfile(ctx context.Context, u *domain.User) error {
	r.users[u.ID] = u
	return nil
}

func (r *testRepo) UpdatePassword(ctx context.Context, id uuid.UUID, hash string) error {
	if u, ok := r.users[id]; ok {
		u.PasswordHash = &hash
		return nil
	}
	return domain.ErrUserNotFound
}

func (r *testRepo) UpdateStatus(ctx context.Context, id uuid.UUID, s domain.Status) error {
	if u, ok := r.users[id]; ok {
		u.Status = s
		return nil
	}
	return domain.ErrUserNotFound
}

func (r *testRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if u, ok := r.users[id]; ok {
		now := time.Now()
		u.DeletedAt = &now
		return nil
	}
	return domain.ErrUserNotFound
}

func (r *testRepo) List(ctx context.Context, f usecase.ListFilter) ([]*domain.User, int64, error) {
	var list []*domain.User
	for _, u := range r.users {
		list = append(list, u)
	}
	return list, int64(len(list)), nil
}

func setupTestServer() (http.Handler, *testRepo) {
	repo := newTestRepo()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	uc := usecase.NewUserUseCase(repo, nil, logger)
	handler := v1.NewHandler(uc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})
	return r, repo
}

func TestHandler_CreateUser(t *testing.T) {
	router, _ := setupTestServer()

	payload := map[string]any{
		"cellphone_number": "+12025550133",
		"first_name":       "Alice",
		"last_name":        "Smith",
		"birthdate":        "1995-05-15",
		"password":         "securePassword123",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var res delivery.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if !res.Success {
		t.Errorf("expected success to be true")
	}
}

func TestHandler_CreateUser_MissingPhone(t *testing.T) {
	router, _ := setupTestServer()

	payload := map[string]any{
		"first_name": "NoPhone",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", rec.Code)
	}
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	router, _ := setupTestServer()

	nonExistentID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+nonExistentID, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 Not Found, got %d", rec.Code)
	}
}
