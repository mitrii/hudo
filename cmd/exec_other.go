//go:build !linux

package cmd

import (
	"context"
	"fmt"
)

func runPrivileged(_ context.Context, _ string) (int, error) {
	return 0, fmt.Errorf("hudo is only supported on Linux")
}
