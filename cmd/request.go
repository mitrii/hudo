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
	Use:     "request <command>",
	Aliases: []string{"r"},
	Short:   "Request privileged execution of a command",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runRequest,
}

func runRequest(cmd *cobra.Command, args []string) error {
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

	_, err = fmt.Fprintln(cmd.OutOrStdout(), shortID(mac))

	return err
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

func formatTTL(seconds int) string {
	d := seconds / 86400
	seconds %= 86400
	h := seconds / 3600
	seconds %= 3600
	m := seconds / 60
	s := seconds % 60

	parts := make([]string, 0, 4)

	if d > 0 {
		parts = append(parts, fmt.Sprintf("%d day", d))
		if d != 1 {
			parts[len(parts)-1] += "s"
		}
	}

	if h > 0 {
		parts = append(parts, fmt.Sprintf("%d hour", h))
		if h != 1 {
			parts[len(parts)-1] += "s"
		}
	}

	if m > 0 {
		parts = append(parts, fmt.Sprintf("%d minute", m))
		if m != 1 {
			parts[len(parts)-1] += "s"
		}
	}

	if s > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d second", s))
		if s != 1 {
			parts[len(parts)-1] += "s"
		}
	}

	return strings.Join(parts, " ")
}

// escapeMD escapes Telegram MarkdownV2 special characters in plain text.
func escapeMD(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`_`, `\_`,
		`*`, `\*`,
		`[`, `\[`,
		`]`, `\]`,
		`(`, `\(`,
		`)`, `\)`,
		`~`, `\~`,
		"`", "\\`",
		`>`, `\>`,
		`#`, `\#`,
		`+`, `\+`,
		`-`, `\-`,
		`=`, `\=`,
		`|`, `\|`,
		`{`, `\{`,
		`}`, `\}`,
		`.`, `\.`,
		`!`, `\!`,
	)

	return replacer.Replace(s)
}

func shortID(mac string) string {
	if len(mac) < 8 {
		return mac
	}

	return mac[:8]
}

func formatMessage(command, mac, pin string, ttl int) string {
	return fmt.Sprintf(
		"hudo request\n\n`%s · %s` \\(expires in %s\\)\n\n```\n%s\n```\n\n||hmac: %s||",
		pin,
		shortID(mac),
		escapeMD(formatTTL(ttl)),
		command,
		mac,
	)
}
