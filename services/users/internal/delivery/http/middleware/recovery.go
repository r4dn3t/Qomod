package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer recovers from panics, logs stack traces, and returns a clean 500 JSON response.
func Recoverer(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					stack := string(debug.Stack())
					logger.Error("panic recovered in http handler",
						"panic", rvr,
						"path", r.URL.Path,
						"stack", stack,
					)

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"success": false,
						"error": map[string]string{
							"code":    "INTERNAL_SERVER_ERROR",
							"message": "an unexpected internal error occurred",
						},
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
