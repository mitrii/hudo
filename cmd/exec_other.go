//go:build !linux

package cmd

import "fmt"

func runPrivileged(_ string) (int, error) {
	return 0, fmt.Errorf("hudo is only supported on Linux")
}
