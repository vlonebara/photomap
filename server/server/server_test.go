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

func TestRun(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	addr, done := startServer(t, ctx, server.NewServer("127.0.0.1:0"))
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()

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

	cancel()
	if err := waitForResult(t, done); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	response, err = client.Get("http://" + addr + "/health")
	if err == nil {
		response.Body.Close()
		t.Fatal("server still accepts requests after shutdown")
	}
}

func TestRunWaitsForActiveRequest(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	requestStarted := make(chan struct{})
	finishRequest := make(chan struct{})
	releaseRequest := sync.OnceFunc(func() { close(finishRequest) })
	defer releaseRequest()

	srv := server.NewServer("127.0.0.1:0")
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		select {
		case <-finishRequest:
			fmt.Fprintln(w, "done")
		case <-r.Context().Done():
		}
	})
	shutdownStarted := make(chan struct{})
	srv.RegisterOnShutdown(func() { close(shutdownStarted) })
	addr, done := startServer(t, ctx, srv)

	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
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

	waitForSignal(t, requestStarted)
	cancel()
	waitForSignal(t, shutdownStarted)

	select {
	case err := <-done:
		t.Fatalf("server exited before active request completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	releaseRequest()
	if err := waitForResult(t, requestErrors); err != nil {
		t.Fatal(err)
	}
	if err := waitForResult(t, done); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestRunListenError(t *testing.T) {
	t.Parallel()

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

func startServer(t *testing.T, ctx context.Context, srv *http.Server) (string, <-chan error) {
	t.Helper()

	addresses := make(chan string, 1)
	srv.BaseContext = func(listener net.Listener) context.Context {
		addresses <- listener.Addr().String()
		return context.Background()
	}
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("close server: %v", err)
		}
	})

	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, srv) }()

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

func waitForSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for signal")
	}
}
