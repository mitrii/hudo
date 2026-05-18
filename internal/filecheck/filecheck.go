// Package filecheck validates ownership and permissions of sensitive files.
package filecheck

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// CheckSafe returns nil if path does not exist (will be created safely by the
// caller) or if it is owned by root and its permissions are no broader than
// maxPerm.  It returns a descriptive error otherwise, matching the behaviour
// of sudo and sshd when they detect unsafe file permissions.
//
// The check is skipped when the environment variable HUDO_SKIP_PERM_CHECK is
// set to "1" — this is intended exclusively for tests running as a non-root
// user.
func CheckSafe(path string, maxPerm fs.FileMode) error {
	if os.Getenv("HUDO_SKIP_PERM_CHECK") == "1" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("stat %s: %w", path, err)
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("stat %s: cannot read ownership info", path)
	}

	if stat.Uid != 0 {
		return fmt.Errorf("bad owner on %s: must be owned by root (uid 0), got uid %d", path, stat.Uid)
	}

	perm := info.Mode().Perm()
	if perm&^maxPerm != 0 {
		return fmt.Errorf("unsafe permissions on %s: got %04o, want at most %04o", path, perm, maxPerm)
	}

	return nil
}
