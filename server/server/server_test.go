//go:build integration

package server_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/vlonebara/photomap/server"
)

// checking server startup and shutdown
func TestRun(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// starting server on a free local port
	addr, done := startServer(t, ctx, server.NewServer("127.0.0.1:0"))
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()

	// checking health response over a real connection
	response, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "{\"status\":\"ok\"}\n" {
		t.Fatalf("health response: status = %d, body = %q", response.StatusCode, body)
	}

	// stopping server and waiting for result
	cancel()
	if err := waitForResult(t, done); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	// checking server no longer accepts connections
	response, err = client.Get("http://" + addr + "/health")
	if err == nil {
		response.Body.Close()
		t.Fatal("server still accepts requests after shutdown")
	}
}

// checking shutdown waits for active request
func TestRunWaitsForActiveRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	// controlling request completion from the test
	requestStarted := make(chan struct{})
	finishRequest := make(chan struct{})
	releaseRequest := sync.OnceFunc(func() { close(finishRequest) })
	defer releaseRequest()

	// keeping request active until the test allows a response
	srv := server.NewServer("127.0.0.1:0")
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		select {
		case <-finishRequest:
			fmt.Fprintln(w, "done")
		case <-r.Context().Done():
		}
	})
	// tracking when shutdown starts
	shutdownStarted := make(chan struct{})
	srv.RegisterOnShutdown(func() { close(shutdownStarted) })
	addr, done := startServer(t, ctx, srv)

	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	// sending request without blocking the test
	requestErrors := make(chan error, 1)
	go func() {
		response, err := client.Get("http://" + addr + "/slow")
		if err == nil {
			defer response.Body.Close()
			body, readErr := io.ReadAll(response.Body)
			err = readErr
			if err == nil && (response.StatusCode != http.StatusOK || string(body) != "done\n") {
				err = fmt.Errorf("active request: status = %d, body = %q", response.StatusCode, body)
			}
		}
		requestErrors <- err
	}()

	// starting shutdown while the request is still active
	waitForSignal(t, requestStarted)
	cancel()
	waitForSignal(t, shutdownStarted)

	// checking server keeps waiting for the active request
	select {
	case err := <-done:
		t.Fatalf("server exited before active request completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	// allowing request to finish before server exits
	releaseRequest()
	if err := waitForResult(t, requestErrors); err != nil {
		t.Fatal(err)
	}
	if err := waitForResult(t, done); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

// checking startup error when port is already in use
func TestRunListenError(t *testing.T) {
	t.Parallel()

	// reserving a local port before starting the server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	done := make(chan error, 1)
	go func() {
		done <- server.Run(t.Context(), server.NewServer(listener.Addr().String()))
	}()

	if err := waitForResult(t, done); !errors.Is(err, syscall.EADDRINUSE) {
		t.Fatalf("error = %v, want address already in use", err)
	}
}

// starting test server and returning its address and result channel
func startServer(t *testing.T, ctx context.Context, srv *http.Server) (string, <-chan error) {
	t.Helper()

	// getting address after the listener is created
	addresses := make(chan string, 1)
	srv.BaseContext = func(listener net.Listener) context.Context {
		addresses <- listener.Addr().String()
		return context.Background()
	}
	// closing server even if the test fails
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	})

	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, srv) }()

	// waiting for startup result with timeout
	select {
	case addr := <-addresses:
		return addr, done
	case err := <-done:
		t.Fatalf("server stopped during startup: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("server startup timed out")
	}
	return "", done
}

// waiting for operation result with timeout
func waitForResult(t *testing.T, results <-chan error) error {
	t.Helper()

	select {
	case err := <-results:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for result")
		return nil
	}
}

// waiting for test synchronization signal with timeout
func waitForSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for signal")
	}
}
