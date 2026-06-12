// Package store persists pending PIN requests using bbolt.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"

	"hudo/internal/filecheck"
)

// ErrNotFound is returned when no pending request matches the given HMAC.
var ErrNotFound = errors.New("pending request not found")

// ErrExpired is returned when the matching request has passed its TTL.
var ErrExpired = errors.New("pending request expired")

var bucket = []byte("pending")

// Entry is a single pending privilege-escalation request.
type Entry struct {
	Command   string    `json:"command"`
	PIN       string    `json:"pin"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Store wraps a bbolt database for pending requests.
type Store struct {
	db *bolt.DB
}

// Open opens (or creates) the bbolt database at path.
func Open(path string) (*Store, error) {
	if err := filecheck.CheckSafe(path, 0600); err != nil {
		return nil, err
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)

		return err
	}); err != nil {
		return nil, fmt.Errorf("init bucket: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Save stores a pending entry keyed by hmac.
func (s *Store) Save(hmac string, e Entry) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(bucket).Put([]byte(hmac), data)
	})
}

// Consume atomically gets and deletes the entry for hmac.
// Returns ErrNotFound or ErrExpired if the entry cannot be used.
func (s *Store) Consume(hmac string) (*Entry, error) {
	var entry Entry

	var expired bool

	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		data := b.Get([]byte(hmac))

		if data == nil {
			return ErrNotFound
		}

		if err := json.Unmarshal(data, &entry); err != nil {
			return fmt.Errorf("decode entry: %w", err)
		}

		if time.Now().After(entry.ExpiresAt) {
			expired = true
		}

		// Always delete — whether valid or expired.
		return b.Delete([]byte(hmac))
	})
	if err != nil {
		return nil, err
	}

	if expired {
		return nil, ErrExpired
	}

	return &entry, nil
}

// Purge removes all expired entries from the store.
func (s *Store) Purge() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		now := time.Now()

		var stale [][]byte

		_ = b.ForEach(func(k, v []byte) error {
			var e Entry
			if err := json.Unmarshal(v, &e); err != nil || now.After(e.ExpiresAt) {
				key := make([]byte, len(k))
				copy(key, k)

				stale = append(stale, key)
			}

			return nil
		})

		for _, k := range stale {
			if err := b.Delete(k); err != nil {
				return err
			}
		}

		return nil
	})
}

// FindPending searches for an unexpired entry matching the given command.
// Returns the HMAC key, the entry, or ErrNotFound if no active entry exists.
func (s *Store) FindPending(command string) (string, *Entry, error) {
	var (
		foundEntry *Entry
		foundKey   string
	)

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		now := time.Now()

		_ = b.ForEach(func(k, v []byte) error {
			if foundEntry != nil {
				return nil // already found
			}

			var e Entry
			if err := json.Unmarshal(v, &e); err == nil {
				if e.Command == command && now.Before(e.ExpiresAt) {
					foundKey = string(k)
					foundEntry = &e
				}
			}

			return nil
		})

		return nil
	})
	if err != nil {
		return "", nil, err
	}

	if foundEntry != nil {
		return foundKey, foundEntry, nil
	}

	return "", nil, ErrNotFound
}
