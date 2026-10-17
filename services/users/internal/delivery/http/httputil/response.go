package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"qomod/services/users/internal/domain"
)

// Response standard envelope for successful JSON responses.
type Response struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
}

// PaginatedResponse envelope for list queries with pagination metadata.
type PaginatedResponse struct {
	Success    bool       `json:"success"`
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination metadata.
type Pagination struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// ErrorResponse standard envelope for failed JSON responses.
type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail describes an API error.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON payload with the specified status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	})
}

// Paginated writes a JSON list response with pagination metadata.
func Paginated(w http.ResponseWriter, status int, data any, total int64, limit, offset int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: Pagination{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Error writes a structured JSON error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// HandleDomainError maps domain errors to appropriate HTTP status codes and error responses.
func HandleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		Error(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrPhoneAlreadyExists):
		Error(w, http.StatusConflict, "PHONE_ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		Error(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", err.Error())
	case errors.Is(err, domain.ErrInvalidPhoneNumber):
		Error(w, http.StatusBadRequest, "INVALID_PHONE_NUMBER", err.Error())
	case errors.Is(err, domain.ErrInvalidStatus):
		Error(w, http.StatusBadRequest, "INVALID_STATUS", err.Error())
	case errors.Is(err, domain.ErrWeakPassword):
		Error(w, http.StatusBadRequest, "WEAK_PASSWORD", err.Error())
	case errors.Is(err, domain.ErrUserDeleted):
		Error(w, http.StatusGone, "USER_DELETED", err.Error())
	case errors.Is(err, domain.ErrUserSuspended):
		Error(w, http.StatusForbidden, "USER_SUSPENDED", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}
