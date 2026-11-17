package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// time limit for active requests during shutdown
const shutdownTimeout = 10 * time.Second

// server initialization
func NewServer(addr string, router http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: router,
		// request headers timeout
		ReadHeaderTimeout: 5 * time.Second,
		// idle connection timeout
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 16 << 10,
	}
}

// starting server and waiting for shutdown
func Run(ctx context.Context, srv *http.Server) error {
	// buffered channel lets the server send its error while shutdown is running
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.ListenAndServe()
	}()

	// waiting for server error or context cancellation
	select {
	case err := <-serverErrors:
		return checkServerError(err)
	case <-ctx.Done():
	}

	// separate context gives active requests time to finish after cancellation
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// stop accepting connections and wait for active requests
	shutdownErr := srv.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		// force closing connections if graceful shutdown fails
		shutdownErr = fmt.Errorf("shutdown HTTP server: %w", errors.Join(shutdownErr, srv.Close()))
	}

	// waiting for server goroutine and returning shutdown errors
	return errors.Join(shutdownErr, checkServerError(<-serverErrors))
}

// checking server errors
func checkServerError(err error) error {
	// normal shutdown is not an error
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("HTTP server: %w", err)
}
