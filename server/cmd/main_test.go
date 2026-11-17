//go:build integration

package main

import (
	"errors"
	"os"
	"testing"
)

// checking startup stops before HTTP when the database is missing
func TestRunMissingDatabase(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := run(); err == nil {
		t.Fatal("server started without the database")
	}
	if _, err := os.Stat("photomap.db"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("startup created an empty database: %v", err)
	}
}
