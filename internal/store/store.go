package store

import (
	"context"
	"sync"
	"time"
)

// Entry represents a single key-value pair in the store,
// optionally with an expiration time.
type Entry struct {
	Value     string
	ExpiresAt time.Time // zero value means no expiration
}

// isExpired reports whether this entry has passed its TTL.
func (e Entry) isExpired() bool {
	return !e.ExpiresAt.IsZero() && time.Now().After(e.ExpiresAt)
}

// Store is a concurrent-safe in-memory key-value store.
// It uses a single RWMutex to protect the underlying map,
// favoring simplicity over sharded-lock complexity.
type Store struct {
	mu   sync.RWMutex
	data map[string]Entry
}

// New creates and returns an empty Store.
func New() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

// Set stores a key-value pair. If ttl > 0, the key will expire
// after the given duration. A ttl of 0 means no expiration.
func (s *Store) Set(key, value string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := Entry{Value: value}
	if ttl > 0 {
		e.ExpiresAt = time.Now().Add(ttl)
	}
	s.data[key] = e
}

// Get retrieves the value for a key. Returns ("", false) if the key
// does not exist or has expired. Expired keys are lazily deleted.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return "", false
	}

	if e.isExpired() {
		// Lazy deletion: upgrade to write lock and remove the expired key.
		// Double-check after acquiring the write lock to avoid races.
		s.mu.Lock()
		if e2, ok2 := s.data[key]; ok2 && e2.isExpired() {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return "", false
	}

	return e.Value, true
}

// Delete removes a key from the store. Returns true if the key existed
// (and was not expired), false otherwise.
func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.data[key]
	if !ok || e.isExpired() {
		// Clean up expired entry if present
		if ok {
			delete(s.data, key)
		}
		return false
	}
	delete(s.data, key)
	return true
}

// StartReaper launches a background goroutine that periodically sweeps
// the store and removes expired keys. It stops when ctx is cancelled.
func (s *Store) StartReaper(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.reap()
			}
		}
	}()
}

// reap removes all expired entries from the store.
func (s *Store) reap() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, e := range s.data {
		if !e.ExpiresAt.IsZero() && now.After(e.ExpiresAt) {
			delete(s.data, key)
		}
	}
}
