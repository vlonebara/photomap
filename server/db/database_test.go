//go:build integration

package db_test

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/vlonebara/photomap/db"
	sqlite3 "modernc.org/sqlite/lib"
)

// checking settings on several physical connections in one store
func TestOpenConnectionSettings(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	if _, err := store.DB.ExecContext(t.Context(), "PRAGMA journal_mode = WAL"); err != nil {
		t.Fatal(err)
	}

	// holding connections open makes the pool create separate connections
	for i := 0; i < 3; i++ {
		connection, err := store.DB.Conn(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := connection.Close(); err != nil {
				t.Errorf("close connection: %v", err)
			}
		})

		settings := []struct {
			name  string
			query string
			want  int
		}{
			{name: "foreign keys", query: "PRAGMA foreign_keys", want: 1},
			{name: "busy timeout", query: "PRAGMA busy_timeout", want: 5000},
			{name: "synchronous", query: "PRAGMA synchronous", want: 2},
		}
		for _, setting := range settings {
			t.Run(fmt.Sprintf("connection %d/%s", i, setting.name), func(t *testing.T) {
				var got int
				if err := connection.QueryRowContext(t.Context(), setting.query).Scan(&got); err != nil {
					t.Fatal(err)
				}
				if got != setting.want {
					t.Errorf("setting = %d, want %d", got, setting.want)
				}
			})
		}

		// opening another connection must preserve the existing journal mode
		var journalMode string
		if err := connection.QueryRowContext(t.Context(), "PRAGMA journal_mode").Scan(&journalMode); err != nil {
			t.Fatal(err)
		}
		if journalMode != "wal" {
			t.Errorf("connection %d: journal mode = %q, want wal", i, journalMode)
		}
	}

	if sqlite3.SQLITE_VERSION_NUMBER < 3051003 {
		t.Fatalf("SQLite %s does not meet the minimum version", sqlite3.SQLITE_VERSION)
	}
	t.Logf("SQLite %s", sqlite3.SQLITE_VERSION)
}

// checking a missing database is not silently created
func TestOpenMissingDatabase(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.db")
	connection, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	if err := connection.PingContext(t.Context()); err == nil {
		t.Fatal("ping succeeded for a missing database")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing database was created: %v", err)
	}
}

// checking an empty path is rejected
func TestOpenEmptyPath(t *testing.T) {
	t.Parallel()

	if connection, err := db.Open(""); err == nil {
		connection.Close()
		t.Fatal("open succeeded with an empty path")
	}
}

// checking the pool cannot be used after shutdown
func TestStoreClose(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	if err := store.DB.PingContext(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := store.DB.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.DB.PingContext(t.Context()); err == nil {
		t.Fatal("ping succeeded after closing the pool")
	}
}

// creating a temporary database without touching photomap.db
func newTestStore(t *testing.T) *db.Store {
	t.Helper()

	// special path characters must remain part of the filename
	path := filepath.Join(t.TempDir(), "photo map?#.db")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	connection, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	closeTestPool(t, connection)
	return db.NewStore(connection)
}

// closing the test pool even if an assertion fails
func closeTestPool(t *testing.T, connection *sql.DB) {
	t.Helper()

	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
}
