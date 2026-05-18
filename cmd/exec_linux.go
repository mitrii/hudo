//go:build linux

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// safeEnv returns a minimal environment stripped of dangerous variables.
var safeEnv = []string{
	"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
}

func runPrivileged(command string) (int, error) {
	if err := syscall.Setgid(0); err != nil {
		return 0, fmt.Errorf("setgid: %w", err)
	}
	if err := syscall.Setuid(0); err != nil {
		return 0, fmt.Errorf("setuid: %w", err)
	}

	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Env = safeEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 0, fmt.Errorf("exec: %w", err)
	}
	return 0, nil
}
