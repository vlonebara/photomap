package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vlonebara/photomap/server"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.NewServer(":8080")
	log.Printf("Starting HTTP server on %s", srv.Addr)

	if err := server.Run(ctx, srv); err != nil {
		return err
	}

	log.Print("HTTP server stopped")
	return nil
}
