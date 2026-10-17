package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"qomod/services/users/internal/delivery/http/httputil"
	"qomod/services/users/internal/domain"
	"qomod/services/users/internal/usecase"
)

// Handler handles HTTP requests for user endpoints.
type Handler struct {
	uc *usecase.UserUseCase
}

// NewHandler creates a new user HTTP handler.
func NewHandler(uc *usecase.UserUseCase) *Handler {
	return &Handler{uc: uc}
}

// RegisterRoutes registers the user API routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/users", func(r chi.Router) {
		r.Post("/", h.CreateUser)
		r.Get("/", h.ListUsers)
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.GetUser)
			r.Put("/profile", h.UpdateProfile)
			r.Put("/password", h.ChangePassword)
			r.Patch("/status", h.UpdateStatus)
			r.Delete("/", h.DeleteUser)
		})
	})
}

// CreateUser handles POST /api/v1/users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON request payload")
		return
	}

	if req.CellphoneNumber == "" {
		httputil.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "cellphone_number is mandatory")
		return
	}

	user, err := h.uc.RegisterUser(r.Context(), req.ToCommand())
	if err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusCreated, ToUserResponse(user))
}

// GetUser handles GET /api/v1/users/{id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_UUID", "id must be a valid UUID")
		return
	}

	user, err := h.uc.GetUserByID(r.Context(), id)
	if err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, ToUserResponse(user))
}

// ListUsers handles GET /api/v1/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(query.Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	var statusFilter *domain.Status
	if s := query.Get("status"); s != "" {
		st := domain.Status(s)
		if st.IsValid() {
			statusFilter = &st
		}
	}

	filter := usecase.ListFilter{
		Limit:  limit,
		Offset: offset,
		Status: statusFilter,
	}

	users, total, err := h.uc.ListUsers(r.Context(), filter)
	if err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	respList := make([]UserResponse, 0, len(users))
	for _, u := range users {
		respList = append(respList, ToUserResponse(u))
	}

	httputil.Paginated(w, http.StatusOK, respList, total, limit, offset)
}

// UpdateProfile handles PUT /api/v1/users/{id}/profile
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_UUID", "id must be a valid UUID")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON request payload")
		return
	}

	user, err := h.uc.UpdateProfile(r.Context(), req.ToCommand(id))
	if err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, ToUserResponse(user))
}

// ChangePassword handles PUT /api/v1/users/{id}/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_UUID", "id must be a valid UUID")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON request payload")
		return
	}

	if req.NewPassword == "" {
		httputil.Error(w, http.StatusBadRequest, "VALIDATION_FAILED", "new_password is required")
		return
	}

	cmd := usecase.ChangePasswordCommand{
		ID:          id,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	if err := h.uc.ChangePassword(r.Context(), cmd); err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "password updated successfully",
	})
}

// UpdateStatus handles PATCH /api/v1/users/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_UUID", "id must be a valid UUID")
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON request payload")
		return
	}

	cmd := usecase.UpdateStatusCommand{
		ID:     id,
		Status: domain.Status(req.Status),
	}

	user, err := h.uc.UpdateStatus(r.Context(), cmd)
	if err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, ToUserResponse(user))
}

// DeleteUser handles DELETE /api/v1/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		httputil.Error(w, http.StatusBadRequest, "INVALID_UUID", "id must be a valid UUID")
		return
	}

	if err := h.uc.DeleteUser(r.Context(), id); err != nil {
		httputil.HandleDomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}
