package filecheck_test

import (
	"os"
	"path/filepath"
	"testing"

	"hudo/internal/filecheck"
)

func TestCheckSafe_NotExist(t *testing.T) {
	err := filecheck.CheckSafe(filepath.Join(t.TempDir(), "nonexistent.db"), 0600)
	if err != nil {
		t.Fatalf("expected nil for non-existent file, got: %v", err)
	}
}

func TestCheckSafe_UnsafePermissions(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "cfg")
	if err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// 0644 is broader than 0600 — group read is not allowed.
	// #nosec G302 — intentionally setting unsafe perms to test the checker.
	if err := os.Chmod(f.Name(), 0644); err != nil {
		t.Fatal(err)
	}

	err = filecheck.CheckSafe(f.Name(), 0600)
	if err == nil {
		t.Fatal("expected error for 0644 permissions with maxPerm 0600, got nil")
	}
}

func TestCheckSafe_StricterPermissionsOK(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping root-owner check: not running as root")
	}

	f, err := os.CreateTemp(t.TempDir(), "cfg")
	if err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(f.Name(), 0400); err != nil {
		t.Fatal(err)
	}

	if err := filecheck.CheckSafe(f.Name(), 0600); err != nil {
		t.Fatalf("expected nil for 0400 with maxPerm 0600, got: %v", err)
	}
}

func TestCheckSafe_BadOwner(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping bad-owner check: running as root (all files owned by root)")
	}

	f, err := os.CreateTemp(t.TempDir(), "cfg")
	if err != nil {
		t.Fatal(err)
	}

	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(f.Name(), 0600); err != nil {
		t.Fatal(err)
	}

	err = filecheck.CheckSafe(f.Name(), 0600)
	if err == nil {
		t.Fatal("expected error for non-root owner, got nil")
	}
}
