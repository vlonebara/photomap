package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vlonebara/photomap/handlers"
)

const shutdownTimeout = 10 * time.Second

func NewServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handlers.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func Run(ctx context.Context, srv *http.Server) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return checkServerError(err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownErr := srv.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		shutdownErr = fmt.Errorf("shutdown HTTP server: %w", errors.Join(shutdownErr, srv.Close()))
	}

	return errors.Join(shutdownErr, checkServerError(<-serverErrors))
}

func checkServerError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("HTTP server: %w", err)
}
