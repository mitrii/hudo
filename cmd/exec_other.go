//go:build !linux

package cmd

import "fmt"

func runPrivileged(_ string) (int, error) {
	return 0, fmt.Errorf("remote-sudo is only supported on Linux")
}
