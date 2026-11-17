package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/vlonebara/photomap/db"
	"github.com/vlonebara/photomap/demo"
)

// importing the demo only when this command is explicitly run
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connection, err := db.Open("photomap.db")
	if err != nil {
		return err
	}

	defer connection.Close()

	if err := connection.PingContext(ctx); err != nil {
		return err
	}

	if err := demo.Seed(ctx, db.NewStore(connection), "media"); err != nil {
		return err
	}

	log.Print("Demonstration users, photos, stars and reports are ready")
	return nil
}
