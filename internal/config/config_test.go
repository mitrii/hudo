package config_test

import (
	"testing"

	"remote-sudo/internal/config"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("REMOTE_SUDO_HMAC_SECRET", "testsecret")
	t.Setenv("REMOTE_SUDO_WEBHOOK_URL", "https://example.com/hook")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.HMACSecret != "testsecret" {
		t.Errorf("HMACSecret: got %q, want %q", cfg.HMACSecret, "testsecret")
	}

	if cfg.WebhookURL != "https://example.com/hook" {
		t.Errorf("WebhookURL: got %q", cfg.WebhookURL)
	}

	if cfg.StorePath != "/var/lib/remote-sudo/pending.db" {
		t.Errorf("StorePath default: got %q", cfg.StorePath)
	}

	if cfg.PINTTLSeconds != 300 {
		t.Errorf("PINTTLSeconds default: got %d", cfg.PINTTLSeconds)
	}
}

func TestLoadMissingSecret(t *testing.T) {
	t.Setenv("REMOTE_SUDO_HMAC_SECRET", "")
	t.Setenv("REMOTE_SUDO_WEBHOOK_URL", "https://example.com/hook")

	_, err := config.Load()
	if err == nil {
		t.Error("expected error when hmac_secret is missing")
	}
}

func TestLoadMissingWebhook(t *testing.T) {
	t.Setenv("REMOTE_SUDO_HMAC_SECRET", "secret")
	t.Setenv("REMOTE_SUDO_WEBHOOK_URL", "")

	_, err := config.Load()
	if err == nil {
		t.Error("expected error when webhook_url is missing")
	}
}

func TestLoadEnvOverridesTTL(t *testing.T) {
	t.Setenv("REMOTE_SUDO_HMAC_SECRET", "s")
	t.Setenv("REMOTE_SUDO_WEBHOOK_URL", "https://x.com")
	t.Setenv("REMOTE_SUDO_PIN_TTL_SECONDS", "60")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.PINTTLSeconds != 60 {
		t.Errorf("PINTTLSeconds: got %d, want 60", cfg.PINTTLSeconds)
	}
}

func TestLoadEnvOverridesStorePath(t *testing.T) {
	t.Setenv("REMOTE_SUDO_HMAC_SECRET", "s")
	t.Setenv("REMOTE_SUDO_WEBHOOK_URL", "https://x.com")
	t.Setenv("REMOTE_SUDO_STORE_PATH", "/tmp/test.db")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.StorePath != "/tmp/test.db" {
		t.Errorf("StorePath: got %q, want /tmp/test.db", cfg.StorePath)
	}
}
