// Package cmd implements the hudo subcommands.
package cmd

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"hudo/internal/config"
	"hudo/internal/notify"
	"hudo/internal/store"
)

// RequestCmd generates a PIN for the given command and sends it via webhook.
var RequestCmd = &cobra.Command{
	Use:   "request <command>",
	Short: "Request privileged execution of a command",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRequest,
}

func runRequest(_ *cobra.Command, args []string) error {
	command := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	s, err := store.Open(cfg.StorePath)
	if err != nil {
		return err
	}

	defer func() {
		_ = s.Close()
	}()

	// Purge stale entries on every request.
	_ = s.Purge()

	pin, err := generatePIN()
	if err != nil {
		return fmt.Errorf("generate pin: %w", err)
	}

	mac := computeHMAC(cfg.HMACSecret, command, pin)

	entry := store.Entry{
		Command:   command,
		PIN:       pin,
		ExpiresAt: time.Now().Add(time.Duration(cfg.PINTTLSeconds) * time.Second),
	}

	if err := s.Save(mac, entry); err != nil {
		return fmt.Errorf("save request: %w", err)
	}

	message := formatMessage(command, mac, pin, cfg.PINTTLSeconds)

	if err := notify.Send(cfg.WebhookURL, message); err != nil {
		return fmt.Errorf("notify: %w", err)
	}

	fmt.Println("OK")

	return nil
}

func generatePIN() (string, error) {
	limit := big.NewInt(1_000_000)

	n, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func computeHMAC(secret, command, pin string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(command + pin))

	return fmt.Sprintf("%x", h.Sum(nil))
}

func formatMessage(command, mac, pin string, ttl int) string {
	return fmt.Sprintf(
		"hudo request\n\nCommand: %s\nHMAC:    %s\nPIN:     %s\n\nExpires in %ds",
		command, mac, pin, ttl,
	)
}
