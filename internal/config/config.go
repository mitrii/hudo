// Package config loads remote-sudo configuration from file and environment.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the runtime configuration for remote-sudo.
type Config struct {
	HMACSecret    string `mapstructure:"hmac_secret"`
	WebhookURL    string `mapstructure:"webhook_url"`
	StorePath     string `mapstructure:"store_path"`
	PINTTLSeconds int    `mapstructure:"pin_ttl_seconds"`
}

// Load reads /etc/remote-sudo/config.yaml and applies REMOTE_SUDO_* env overrides.
func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/remote-sudo")

	v.SetDefault("store_path", "/var/lib/remote-sudo/pending.db")
	v.SetDefault("pin_ttl_seconds", 300)

	// Bind each key explicitly so env overrides work regardless of config file presence.
	_ = v.BindEnv("hmac_secret", "REMOTE_SUDO_HMAC_SECRET")
	_ = v.BindEnv("webhook_url", "REMOTE_SUDO_WEBHOOK_URL")
	_ = v.BindEnv("store_path", "REMOTE_SUDO_STORE_PATH")
	_ = v.BindEnv("pin_ttl_seconds", "REMOTE_SUDO_PIN_TTL_SECONDS")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.HMACSecret == "" {
		return nil, fmt.Errorf("hmac_secret is required")
	}

	if cfg.WebhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required")
	}

	return &cfg, nil
}
