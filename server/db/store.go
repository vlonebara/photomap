package db

import "database/sql"

// shared database store for future services and handlers
type Store struct {
	DB *sql.DB
}

// creating store with an already opened connection pool
func NewStore(connection *sql.DB) *Store {
	return &Store{DB: connection}
}
