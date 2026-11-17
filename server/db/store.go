package db

import (
	"context"
	"database/sql"
	"sync"
)

// shared database store for future services and handlers
type Store struct {
	DB *sql.DB
	*Queries
	writeMu sync.Mutex
}

// keeping related writes in one transaction
func (s *Store) WithTx(ctx context.Context, apply func(*Queries) error) error {
	// serialize application writes before SQLite acquires its write lock
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := apply(s.Queries.WithTx(tx)); err != nil {
		return err
	}

	return tx.Commit()
}

// creating store with an already opened connection pool
func NewStore(connection *sql.DB) *Store {
	return &Store{DB: connection, Queries: New(connection)}
}
