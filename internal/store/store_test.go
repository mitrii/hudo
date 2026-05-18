package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hudo/internal/store"
)

func openTemp(t *testing.T) *store.Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")

	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return s
}

func TestSaveAndConsume(t *testing.T) {
	s := openTemp(t)

	entry := store.Entry{
		Command:   "echo hello",
		PIN:       "123456",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	if err := s.Save("hmac1", entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Consume("hmac1")
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}

	if got.Command != entry.Command {
		t.Errorf("Command: got %q, want %q", got.Command, entry.Command)
	}

	if got.PIN != entry.PIN {
		t.Errorf("PIN: got %q, want %q", got.PIN, entry.PIN)
	}
}

func TestConsumeIsOneTime(t *testing.T) {
	s := openTemp(t)

	entry := store.Entry{
		Command:   "ls",
		PIN:       "000001",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	if err := s.Save("hmac2", entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := s.Consume("hmac2"); err != nil {
		t.Fatalf("first Consume: %v", err)
	}

	_, err := s.Consume("hmac2")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("second Consume: got %v, want ErrNotFound", err)
	}
}

func TestConsumeNotFound(t *testing.T) {
	s := openTemp(t)

	_, err := s.Consume("nonexistent")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestConsumeExpired(t *testing.T) {
	s := openTemp(t)

	entry := store.Entry{
		Command:   "id",
		PIN:       "999999",
		ExpiresAt: time.Now().Add(-time.Second), // already expired
	}

	if err := s.Save("hmac3", entry); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err := s.Consume("hmac3")
	if !errors.Is(err, store.ErrExpired) {
		t.Errorf("got %v, want ErrExpired", err)
	}

	// Expired entry is deleted on first Consume — second call returns ErrNotFound.
	_, err = s.Consume("hmac3")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("after expiry delete: got %v, want ErrNotFound", err)
	}
}

func TestPurge(t *testing.T) {
	s := openTemp(t)

	expired := store.Entry{
		Command:   "old",
		PIN:       "111111",
		ExpiresAt: time.Now().Add(-time.Second),
	}

	fresh := store.Entry{
		Command:   "new",
		PIN:       "222222",
		ExpiresAt: time.Now().Add(time.Minute),
	}

	if err := s.Save("expired", expired); err != nil {
		t.Fatalf("Save expired: %v", err)
	}

	if err := s.Save("fresh", fresh); err != nil {
		t.Fatalf("Save fresh: %v", err)
	}

	if err := s.Purge(); err != nil {
		t.Fatalf("Purge: %v", err)
	}

	// Fresh entry must still be consumable.
	if _, err := s.Consume("fresh"); err != nil {
		t.Errorf("fresh entry missing after Purge: %v", err)
	}

	// Expired entry must be gone.
	_, err := s.Consume("expired")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired entry not purged: got %v, want ErrNotFound", err)
	}
}

func TestOpenInvalidPath(t *testing.T) {
	_, err := store.Open(filepath.Join(t.TempDir(), "no", "such", "dir", "db"))
	if err == nil {
		t.Error("expected error opening db in nonexistent directory")
	}
}

func TestOpenCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "created.db")

	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	_ = s.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("db file not created: %v", err)
	}
}
