// Package cmd implements the hudo subcommands.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"hudo/internal/config"
	"hudo/internal/store"
)

var pinFlag string

// ExecCmd verifies the PIN and executes the requested command as root.
var ExecCmd = &cobra.Command{
	Use:   "exec <command>",
	Short: "Execute a previously requested command after PIN verification",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExec,
}

func init() {
	ExecCmd.Flags().StringVar(&pinFlag, "pin", "", "PIN received via notification (required)")
	_ = ExecCmd.MarkFlagRequired("pin")
}

func runExec(_ *cobra.Command, args []string) error {
	command := strings.Join(args, " ")
	pin := pinFlag

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	s, err := store.Open(cfg.StorePath)
	if err != nil {
		return err
	}

	mac := computeHMAC(cfg.HMACSecret, command, pin)

	entry, err := s.Consume(mac)

	if cerr := s.Close(); cerr != nil && err == nil {
		err = cerr
	}

	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Defence-in-depth: command in store must match exactly.
	if entry.Command != command {
		return fmt.Errorf("verification failed: command mismatch")
	}

	exitCode, err := runPrivileged(command)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}
