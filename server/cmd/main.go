package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vlonebara/photomap/db"
	"github.com/vlonebara/photomap/server"
)

// application entry point
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

// server startup and shutdown
func run() (runErr error) {
	// listening for Ctrl+C and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// opening database once and sharing its pool through the store
	connection, err := db.Open("photomap.db")
	if err != nil {
		return err
	}
	store := db.NewStore(connection)

	// closing database after the server has finished its active requests
	defer func() {
		if err := store.DB.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close database: %w", err))
		}
	}()

	// checking connection before starting the HTTP server
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = store.DB.PingContext(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	log.Print("Database connected")

	// server initialization and startup
	srv := server.NewServer(":8080")
	log.Printf("Starting HTTP server on %s", srv.Addr)

	if err := server.Run(ctx, srv); err != nil {
		return err
	}

	log.Print("HTTP server stopped")
	return nil
}
