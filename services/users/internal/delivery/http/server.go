package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server encapsulates the http.Server lifecycle.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer builds an http.Server configured with hardened production timeouts.
func NewServer(port string, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		logger: logger,
		httpServer: &http.Server{
			Addr:              ":" + port,
			Handler:           handler,
			ReadTimeout:       10 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

// Start launches the HTTP server in a non-blocking background goroutine.
func (s *Server) Start() <-chan error {
	errChan := make(chan error, 1)

	go func() {
		s.logger.Info("http server listening", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server failure: %w", err)
		}
		close(errChan)
	}()

	return errChan
}

// Shutdown gracefully drains active connections up to the specified timeout.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server...")
	return s.httpServer.Shutdown(ctx)
}
