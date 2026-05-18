// Package config loads hudo configuration from file and environment.
package config

import (
	"fmt"

	"github.com/spf13/viper"

	"hudo/internal/filecheck"
)

// Config holds the runtime configuration for hudo.
type Config struct {
	HMACSecret    string `mapstructure:"hmac_secret"`
	WebhookURL    string `mapstructure:"webhook_url"`
	StorePath     string `mapstructure:"store_path"`
	PINTTLSeconds int    `mapstructure:"pin_ttl_seconds"`
}

// Load reads /etc/hudo/config.yaml and applies HUDO_* env overrides.
func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/hudo")

	v.SetDefault("store_path", "/var/lib/hudo/pending.db")
	v.SetDefault("pin_ttl_seconds", 300)

	// Bind each key explicitly so env overrides work regardless of config file presence.
	_ = v.BindEnv("hmac_secret", "HUDO_HMAC_SECRET")
	_ = v.BindEnv("webhook_url", "HUDO_WEBHOOK_URL")
	_ = v.BindEnv("store_path", "HUDO_STORE_PATH")
	_ = v.BindEnv("pin_ttl_seconds", "HUDO_PIN_TTL_SECONDS")

	if err := filecheck.CheckSafe("/etc/hudo/config.yaml", 0600); err != nil {
		return nil, err
	}

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
