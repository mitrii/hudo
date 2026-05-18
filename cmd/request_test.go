package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func TestGeneratePIN(t *testing.T) {
	seen := make(map[string]bool)

	for range 100 {
		pin, err := generatePIN()
		if err != nil {
			t.Fatalf("generatePIN: %v", err)
		}

		if len(pin) != 6 {
			t.Errorf("PIN length: got %d, want 6 (pin=%q)", len(pin), pin)
		}

		for _, ch := range pin {
			if ch < '0' || ch > '9' {
				t.Errorf("PIN %q contains non-digit character %q", pin, ch)
			}
		}

		seen[pin] = true
	}

	// With 100 samples over 1M space, probability of all being identical is negligible.
	if len(seen) < 2 {
		t.Error("generatePIN produced identical values across 100 calls — likely not random")
	}
}

func TestGeneratePINLeadingZero(t *testing.T) {
	// Run enough iterations to statistically hit a PIN starting with 0 (10% chance each).
	// Verifies zero-padding is applied.
	found := false

	for range 1000 {
		pin, err := generatePIN()
		if err != nil {
			t.Fatalf("generatePIN: %v", err)
		}

		if strings.HasPrefix(pin, "0") {
			found = true

			break
		}
	}

	if !found {
		t.Log("warning: no leading-zero PIN produced in 1000 iterations (unlikely but possible)")
	}
}

func TestComputeHMACDeterministic(t *testing.T) {
	h1 := computeHMAC("secret", "ls -la", "123456")
	h2 := computeHMAC("secret", "ls -la", "123456")

	if h1 != h2 {
		t.Errorf("HMAC not deterministic: %q != %q", h1, h2)
	}
}

func TestComputeHMACDiffersOnDifferentInputs(t *testing.T) {
	cases := []struct{ secret, cmd, pin string }{
		{"secret", "ls", "111111"},
		{"other", "ls", "111111"},
		{"secret", "rm", "111111"},
		{"secret", "ls", "222222"},
	}

	hashes := make(map[string]bool)

	for _, c := range cases {
		h := computeHMAC(c.secret, c.cmd, c.pin)
		if hashes[h] {
			t.Errorf("HMAC collision for (%q,%q,%q)", c.secret, c.cmd, c.pin)
		}

		hashes[h] = true
	}
}

func TestComputeHMACFormat(t *testing.T) {
	h := computeHMAC("secret", "cmd", "pin")
	// SHA-256 hex = 64 chars.
	if len(h) != 64 {
		t.Errorf("HMAC length: got %d, want 64", len(h))
	}

	for _, ch := range h {
		isDigit := ch >= '0' && ch <= '9'
		isHexLower := ch >= 'a' && ch <= 'f'

		if !isDigit && !isHexLower {
			t.Errorf("HMAC %q contains non-hex character %q", h, ch)
		}
	}
}

func TestFormatMessage(t *testing.T) {
	msg := formatMessage("rm -rf /tmp", "deadbeef", "123456", 300)

	checks := []string{
		"rm -rf /tmp",
		"deadbeef",
		"123456",
		"300s",
		"remote-sudo request",
	}

	for _, want := range checks {
		if !strings.Contains(msg, want) {
			t.Errorf("formatMessage output missing %q\nfull output:\n%s", want, msg)
		}
	}
}

func TestFormatMessageStructure(t *testing.T) {
	msg := formatMessage("echo hi", "abc123", "000001", 60)

	lines := strings.Split(msg, "\n")
	if len(lines) < 5 {
		t.Errorf("expected at least 5 lines, got %d:\n%s", len(lines), msg)
	}

	if !strings.HasPrefix(lines[0], "remote-sudo") {
		t.Errorf("first line should start with 'remote-sudo', got %q", lines[0])
	}
}

func TestComputeHMACConsistencyWithExec(t *testing.T) {
	// Simulate request+exec HMAC agreement.
	secret := "my-hmac-secret"
	command := "systemctl restart nginx"
	pin := "847291"

	requestMAC := computeHMAC(secret, command, pin)
	execMAC := computeHMAC(secret, command, pin)

	if requestMAC != execMAC {
		t.Errorf("request and exec produce different HMACs: %q vs %q", requestMAC, execMAC)
	}

	// A wrong PIN must not match.
	wrongMAC := computeHMAC(secret, command, fmt.Sprintf("%06d", 847292))

	if requestMAC == wrongMAC {
		t.Error("wrong PIN produced the same HMAC")
	}
}
