package db

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// minimum sqlite version with the WAL reset fix
const minimumSQLiteVersion = 3051003

// opening a shared connection pool for an existing database
func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("database path is empty")
	}

	if sqlite3.SQLITE_VERSION_NUMBER < minimumSQLiteVersion {
		return nil, fmt.Errorf("SQLite %s is too old: version 3.51.3 or newer is required", sqlite3.SQLITE_VERSION)
	}

	// absolute path keeps all connections pointed at the same file
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("database path: %w", err)
	}

	// driver applies these settings to every connection in the pool
	params := url.Values{}
	params.Set("mode", "rw")
	params.Add("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "foreign_keys(ON)")
	params.Add("_pragma", "synchronous(FULL)")
	dsn := url.URL{Scheme: "file", Path: absPath, RawQuery: params.Encode()}

	// read-write mode prevents creating an empty database at a wrong path
	connection, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return connection, nil
}
