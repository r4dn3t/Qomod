package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"qomod/services/users/internal/delivery/http/httputil"
	"qomod/services/users/internal/delivery/http/middleware"
	v1 "qomod/services/users/internal/delivery/http/v1"
	"qomod/services/users/internal/usecase"
)

// RouterConfig options for configuring the HTTP router.
type RouterConfig struct {
	Logger  *slog.Logger
	Pool    *pgxpool.Pool
	UseCase *usecase.UserUseCase
}

// NewRouter constructs and configures the main Chi HTTP handler.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Standard middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(middleware.Recoverer(cfg.Logger))
	r.Use(middleware.CORS())
	r.Use(chimiddleware.Timeout(60 * time.Second))

	// Liveness check (indicates process is running)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness check (indicates service can talk to dependencies like Postgres)
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := cfg.Pool.Ping(ctx); err != nil {
			httputil.Error(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "database ping failed")
			return
		}

		httputil.JSON(w, http.StatusOK, map[string]string{
			"status":   "ready",
			"database": "connected",
		})
	})

	// API v1 sub-router
	v1Handler := v1.NewHandler(cfg.UseCase)
	r.Route("/api/v1", func(r chi.Router) {
		v1Handler.RegisterRoutes(r)
	})

	return r
}
