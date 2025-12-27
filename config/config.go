// Package config provides configuration loading and validation.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure.
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Upstream  UpstreamConfig   `yaml:"upstream"`
	Auth      AuthConfig       `yaml:"auth"`
	RateLimit RateLimitConfig  `yaml:"rate_limit"`
	Usage     UsageConfig      `yaml:"usage"`
	Billing   BillingConfig    `yaml:"billing"`
	Database  DatabaseConfig   `yaml:"database"`
	Plans     []PlanConfig     `yaml:"plans"`
	Endpoints []EndpointConfig `yaml:"endpoints"`
	Logging   LoggingConfig    `yaml:"logging"`
	Metrics   MetricsConfig    `yaml:"metrics"`
}

// ServerConfig configures the HTTP server.
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// UpstreamConfig configures the upstream service.
type UpstreamConfig struct {
	URL             string        `yaml:"url"`
	Timeout         time.Duration `yaml:"timeout"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	IdleConnTimeout time.Duration `yaml:"idle_conn_timeout"`
}

// AuthConfig configures authentication.
// Use "local" for built-in auth or "remote" to delegate to external service.
type AuthConfig struct {
	Mode      string       `yaml:"mode"` // "local" or "remote"
	KeyPrefix string       `yaml:"key_prefix"`
	Remote    RemoteConfig `yaml:"remote,omitempty"`
}

// RateLimitConfig configures rate limiting.
type RateLimitConfig struct {
	Enabled     bool `yaml:"enabled"`
	BurstTokens int  `yaml:"burst_tokens"`
	WindowSecs  int  `yaml:"window_secs"`
}

// UsageConfig configures usage tracking.
// Use "local" for built-in storage or "remote" to send to external service.
type UsageConfig struct {
	Mode          string        `yaml:"mode"` // "local" or "remote"
	BatchSize     int           `yaml:"batch_size"`
	FlushInterval time.Duration `yaml:"flush_interval"`
	Remote        RemoteConfig  `yaml:"remote,omitempty"`
}

// BillingConfig configures billing.
// Use "none", "stripe", "paddle", "lemonsqueezy", or "remote".
type BillingConfig struct {
	Mode           string       `yaml:"mode"` // "none", "stripe", "paddle", "lemonsqueezy", "remote"
	StripeKey      string       `yaml:"stripe_key,omitempty"`
	PaddleVendorID string       `yaml:"paddle_vendor_id,omitempty"`
	PaddleAPIKey   string       `yaml:"paddle_api_key,omitempty"`
	LemonAPIKey    string       `yaml:"lemon_api_key,omitempty"`
	Remote         RemoteConfig `yaml:"remote,omitempty"`
}

// RemoteConfig configures a remote service endpoint.
type RemoteConfig struct {
	URL     string            `yaml:"url"`
	APIKey  string            `yaml:"api_key,omitempty"`
	Timeout time.Duration     `yaml:"timeout,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

// DatabaseConfig configures the database.
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // "sqlite" or "postgres" (future)
	DSN    string `yaml:"dsn"`
}

// PlanConfig configures a subscription plan.
type PlanConfig struct {
	ID                 string `yaml:"id"`
	Name               string `yaml:"name"`
	RateLimitPerMinute int    `yaml:"rate_limit_per_minute"`
	RequestsPerMonth   int64  `yaml:"requests_per_month"`
	PriceMonthly       int64  `yaml:"price_monthly"` // cents
	OveragePrice       int64  `yaml:"overage_price"` // cents per request
}

// EndpointConfig configures custom pricing for specific endpoints.
type EndpointConfig struct {
	Method         string  `yaml:"method"`
	Path           string  `yaml:"path"`
	CostMultiplier float64 `yaml:"cost_multiplier"`
}

// LoggingConfig configures logging.
type LoggingConfig struct {
	Level  string `yaml:"level"`  // "debug", "info", "warn", "error"
	Format string `yaml:"format"` // "json" or "console"
}

// MetricsConfig configures Prometheus metrics.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"` // Enable /metrics endpoint
	Path    string `yaml:"path"`    // Custom path (default: /metrics)
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	setDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 60 * time.Second
	}

	if cfg.Auth.Mode == "" {
		cfg.Auth.Mode = "local"
	}
	if cfg.Auth.KeyPrefix == "" {
		cfg.Auth.KeyPrefix = "ak_"
	}

	if cfg.RateLimit.BurstTokens == 0 {
		cfg.RateLimit.BurstTokens = 5
	}
	if cfg.RateLimit.WindowSecs == 0 {
		cfg.RateLimit.WindowSecs = 60
	}

	if cfg.Usage.Mode == "" {
		cfg.Usage.Mode = "local"
	}
	if cfg.Usage.BatchSize == 0 {
		cfg.Usage.BatchSize = 100
	}
	if cfg.Usage.FlushInterval == 0 {
		cfg.Usage.FlushInterval = 10 * time.Second
	}

	if cfg.Billing.Mode == "" {
		cfg.Billing.Mode = "none"
	}

	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = "apigate.db"
	}

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}

	// Default free plan if none configured
	if len(cfg.Plans) == 0 {
		cfg.Plans = []PlanConfig{
			{
				ID:                 "free",
				Name:               "Free",
				RateLimitPerMinute: 60,
				RequestsPerMonth:   1000,
			},
		}
	}
}

func validate(cfg *Config) error {
	if cfg.Upstream.URL == "" {
		return fmt.Errorf("upstream.url is required")
	}

	validAuthModes := map[string]bool{"local": true, "remote": true}
	if !validAuthModes[cfg.Auth.Mode] {
		return fmt.Errorf("auth.mode must be 'local' or 'remote', got %q", cfg.Auth.Mode)
	}
	if cfg.Auth.Mode == "remote" && cfg.Auth.Remote.URL == "" {
		return fmt.Errorf("auth.remote.url is required when auth.mode is 'remote'")
	}

	validUsageModes := map[string]bool{"local": true, "remote": true}
	if !validUsageModes[cfg.Usage.Mode] {
		return fmt.Errorf("usage.mode must be 'local' or 'remote', got %q", cfg.Usage.Mode)
	}
	if cfg.Usage.Mode == "remote" && cfg.Usage.Remote.URL == "" {
		return fmt.Errorf("usage.remote.url is required when usage.mode is 'remote'")
	}

	validBillingModes := map[string]bool{
		"none": true, "stripe": true, "paddle": true, "lemonsqueezy": true, "remote": true,
	}
	if !validBillingModes[cfg.Billing.Mode] {
		return fmt.Errorf("billing.mode must be one of: none, stripe, paddle, lemonsqueezy, remote")
	}
	if cfg.Billing.Mode == "remote" && cfg.Billing.Remote.URL == "" {
		return fmt.Errorf("billing.remote.url is required when billing.mode is 'remote'")
	}

	for i, plan := range cfg.Plans {
		if plan.ID == "" {
			return fmt.Errorf("plans[%d].id is required", i)
		}
	}

	return nil
}
