package service

import (
	"context"
	"sync"
)

// latest public revision sent to connected clients
type MapEvent struct {
	Revision int64 `json:"revision"`
}

type Events struct {
	mu       sync.Mutex
	revision int64
	clients  map[chan MapEvent]struct{}
	done     chan struct{}
}

// closing event streams when the application stops
func NewEvents(ctx context.Context, revision int64) *Events {
	e := &Events{
		revision: revision,
		clients:  make(map[chan MapEvent]struct{}),
		done:     make(chan struct{}),
	}

	go func() {
		select {
		case <-ctx.Done():
			e.Close()
		case <-e.done:
		}
	}()

	return e
}

// a new connection always receives the current revision
func (e *Events) Subscribe() (<-chan MapEvent, func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	client := make(chan MapEvent, 1)
	select {
	case <-e.done:
		close(client)
	default:
		client <- MapEvent{Revision: e.revision}
		e.clients[client] = struct{}{}
	}

	return client, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		if _, ok := e.clients[client]; ok {
			delete(e.clients, client)
			close(client)
		}
	}
}

// replacing stale notifications without blocking writers
func (e *Events) Publish(revision int64) {
	if e == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if revision <= e.revision {
		return
	}

	e.revision = revision
	for client := range e.clients {
		select {
		case <-client:
		default:
		}

		client <- MapEvent{Revision: revision}
	}
}

// releasing subscribers during graceful shutdown
func (e *Events) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	select {
	case <-e.done:
		return
	default:
	}

	close(e.done)
	for client := range e.clients {
		close(client)
		delete(e.clients, client)
	}
}
