// Package config provides configuration loading and validation.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	OpenAPI   OpenAPIConfig    `yaml:"openapi"`
	Portal    PortalConfig     `yaml:"portal"`
	Email     EmailConfig      `yaml:"email"`
	TLS       TLSConfig        `yaml:"tls"`
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
	Header    string       `yaml:"header"` // Header name for API key (default: X-API-Key)
	JWTSecret string       `yaml:"jwt_secret,omitempty"` // Secret for JWT signing (web UI auth)
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

// OpenAPIConfig configures OpenAPI/Swagger documentation.
type OpenAPIConfig struct {
	Enabled bool `yaml:"enabled"` // Enable OpenAPI endpoints
}

// PortalConfig configures the user self-service portal.
type PortalConfig struct {
	Enabled bool   `yaml:"enabled"` // Enable user portal
	BaseURL string `yaml:"base_url"` // Base URL for email links (e.g., https://api.example.com)
	AppName string `yaml:"app_name"` // Application name shown in portal
}

// EmailConfig configures email sending.
type EmailConfig struct {
	Provider   string          `yaml:"provider"` // "smtp", "mock", or "none"
	SMTP       SMTPConfig      `yaml:"smtp,omitempty"`
}

// SMTPConfig configures SMTP email sending.
type SMTPConfig struct {
	Host        string        `yaml:"host"`
	Port        int           `yaml:"port"`
	Username    string        `yaml:"username"`
	Password    string        `yaml:"password"`
	From        string        `yaml:"from"`         // Sender email
	FromName    string        `yaml:"from_name"`    // Sender display name
	UseTLS      bool          `yaml:"use_tls"`
	UseImplicit bool          `yaml:"use_implicit"` // Implicit TLS (port 465)
	SkipVerify  bool          `yaml:"skip_verify"`  // Skip TLS cert verification
	Timeout     time.Duration `yaml:"timeout"`
}

// TLSConfig configures TLS/HTTPS for the server.
// TLS can be configured via ACME (Let's Encrypt) or manual certificate files.
type TLSConfig struct {
	Enabled      bool   `yaml:"enabled"`       // Enable TLS/HTTPS
	Mode         string `yaml:"mode"`          // "acme", "manual", or "none"
	Domain       string `yaml:"domain"`        // Domain(s) for ACME (comma-separated)
	Email        string `yaml:"email"`         // Contact email for ACME
	CertPath     string `yaml:"cert_path"`     // Certificate file path (manual mode)
	KeyPath      string `yaml:"key_path"`      // Private key file path (manual mode)
	HTTPRedirect bool   `yaml:"http_redirect"` // Redirect HTTP to HTTPS
	MinVersion   string `yaml:"min_version"`   // Minimum TLS version: "1.2" or "1.3"
	ACMEStaging  bool   `yaml:"acme_staging"`  // Use Let's Encrypt staging server
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

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	setDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// LoadFromEnv creates configuration entirely from environment variables.
// This is useful for Docker deployments where no config file is needed.
//
// Environment variables:
//
//	APIGATE_UPSTREAM_URL       - Upstream API URL (required)
//	APIGATE_DATABASE_DSN       - Database path (default: apigate.db)
//	APIGATE_SERVER_HOST        - Server host (default: 0.0.0.0)
//	APIGATE_SERVER_PORT        - Server port (default: 8080)
//	APIGATE_AUTH_MODE          - Auth mode: local or remote (default: local)
//	APIGATE_AUTH_KEY_PREFIX    - API key prefix (default: ak_)
//	APIGATE_RATELIMIT_ENABLED  - Enable rate limiting (default: true)
//	APIGATE_LOG_LEVEL          - Log level: debug, info, warn, error (default: info)
//	APIGATE_LOG_FORMAT         - Log format: json or console (default: json)
//	APIGATE_METRICS_ENABLED    - Enable /metrics endpoint (default: true)
//	APIGATE_OPENAPI_ENABLED    - Enable OpenAPI/Swagger (default: true)
//	APIGATE_ADMIN_EMAIL        - Admin email for first-run bootstrap
//	APIGATE_TLS_ENABLED        - Enable TLS/HTTPS (default: false)
//	APIGATE_TLS_MODE           - TLS mode: none, acme, manual (default: none)
//	APIGATE_TLS_DOMAIN         - Domain(s) for ACME (comma-separated)
//	APIGATE_TLS_EMAIL          - Contact email for ACME
//	APIGATE_TLS_CERT           - Certificate file path (manual mode)
//	APIGATE_TLS_KEY            - Private key file path (manual mode)
//	APIGATE_TLS_HTTP_REDIRECT  - Redirect HTTP to HTTPS (default: true when TLS enabled)
//	APIGATE_TLS_MIN_VERSION    - Minimum TLS version: 1.2, 1.3 (default: 1.2)
//	APIGATE_TLS_ACME_STAGING   - Use Let's Encrypt staging (default: false)
func LoadFromEnv() (*Config, error) {
	var cfg Config

	applyEnvOverrides(&cfg)
	setDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

// LoadWithFallback tries to load from file, falls back to environment variables.
// This is the recommended method for Docker deployments.
func LoadWithFallback(path string) (*Config, error) {
	// Try loading from file first
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	// Check if we have enough env vars to run
	if os.Getenv("APIGATE_UPSTREAM_URL") != "" {
		return LoadFromEnv()
	}

	// No config available
	return nil, fmt.Errorf("no configuration found: provide config file or set APIGATE_UPSTREAM_URL")
}

// HasEnvConfig returns true if essential environment variables are set.
func HasEnvConfig() bool {
	return os.Getenv("APIGATE_UPSTREAM_URL") != ""
}

// applyEnvOverrides applies APIGATE_* environment variables to the config.
// Environment variables always override file-based configuration.
func applyEnvOverrides(cfg *Config) {
	// Server configuration
	if v := os.Getenv("APIGATE_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("APIGATE_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("APIGATE_SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("APIGATE_SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}

	// Upstream configuration
	if v := os.Getenv("APIGATE_UPSTREAM_URL"); v != "" {
		cfg.Upstream.URL = v
	}
	if v := os.Getenv("APIGATE_UPSTREAM_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Upstream.Timeout = d
		}
	}

	// Auth configuration
	if v := os.Getenv("APIGATE_AUTH_MODE"); v != "" {
		cfg.Auth.Mode = v
	}
	if v := os.Getenv("APIGATE_AUTH_KEY_PREFIX"); v != "" {
		cfg.Auth.KeyPrefix = v
	}
	if v := os.Getenv("APIGATE_AUTH_REMOTE_URL"); v != "" {
		cfg.Auth.Remote.URL = v
	}

	// Rate limit configuration
	if v := os.Getenv("APIGATE_RATELIMIT_ENABLED"); v != "" {
		cfg.RateLimit.Enabled = parseBool(v)
	}
	if v := os.Getenv("APIGATE_RATELIMIT_BURST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimit.BurstTokens = n
		}
	}
	if v := os.Getenv("APIGATE_RATELIMIT_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimit.WindowSecs = n
		}
	}

	// Usage configuration
	if v := os.Getenv("APIGATE_USAGE_MODE"); v != "" {
		cfg.Usage.Mode = v
	}
	if v := os.Getenv("APIGATE_USAGE_REMOTE_URL"); v != "" {
		cfg.Usage.Remote.URL = v
	}

	// Billing configuration
	if v := os.Getenv("APIGATE_BILLING_MODE"); v != "" {
		cfg.Billing.Mode = v
	}
	if v := os.Getenv("APIGATE_BILLING_STRIPE_KEY"); v != "" {
		cfg.Billing.StripeKey = v
	}

	// Database configuration
	if v := os.Getenv("APIGATE_DATABASE_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("APIGATE_DATABASE_DSN"); v != "" {
		cfg.Database.DSN = v
	}

	// Logging configuration
	if v := os.Getenv("APIGATE_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("APIGATE_LOG_FORMAT"); v != "" {
		cfg.Logging.Format = v
	}

	// Metrics configuration
	if v := os.Getenv("APIGATE_METRICS_ENABLED"); v != "" {
		cfg.Metrics.Enabled = parseBool(v)
	}
	if v := os.Getenv("APIGATE_METRICS_PATH"); v != "" {
		cfg.Metrics.Path = v
	}

	// OpenAPI configuration
	if v := os.Getenv("APIGATE_OPENAPI_ENABLED"); v != "" {
		cfg.OpenAPI.Enabled = parseBool(v)
	}

	// Portal configuration
	if v := os.Getenv("APIGATE_PORTAL_ENABLED"); v != "" {
		cfg.Portal.Enabled = parseBool(v)
	}
	if v := os.Getenv("APIGATE_PORTAL_BASE_URL"); v != "" {
		cfg.Portal.BaseURL = v
	}
	if v := os.Getenv("APIGATE_PORTAL_APP_NAME"); v != "" {
		cfg.Portal.AppName = v
	}

	// Email configuration
	if v := os.Getenv("APIGATE_EMAIL_PROVIDER"); v != "" {
		cfg.Email.Provider = v
	}
	if v := os.Getenv("APIGATE_SMTP_HOST"); v != "" {
		cfg.Email.SMTP.Host = v
	}
	if v := os.Getenv("APIGATE_SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Email.SMTP.Port = port
		}
	}
	if v := os.Getenv("APIGATE_SMTP_USERNAME"); v != "" {
		cfg.Email.SMTP.Username = v
	}
	if v := os.Getenv("APIGATE_SMTP_PASSWORD"); v != "" {
		cfg.Email.SMTP.Password = v
	}
	if v := os.Getenv("APIGATE_SMTP_FROM"); v != "" {
		cfg.Email.SMTP.From = v
	}
	if v := os.Getenv("APIGATE_SMTP_FROM_NAME"); v != "" {
		cfg.Email.SMTP.FromName = v
	}
	if v := os.Getenv("APIGATE_SMTP_USE_TLS"); v != "" {
		cfg.Email.SMTP.UseTLS = parseBool(v)
	}

	// TLS configuration
	if v := os.Getenv("APIGATE_TLS_ENABLED"); v != "" {
		cfg.TLS.Enabled = parseBool(v)
	}
	if v := os.Getenv("APIGATE_TLS_MODE"); v != "" {
		cfg.TLS.Mode = v
	}
	if v := os.Getenv("APIGATE_TLS_DOMAIN"); v != "" {
		cfg.TLS.Domain = v
	}
	if v := os.Getenv("APIGATE_TLS_EMAIL"); v != "" {
		cfg.TLS.Email = v
	}
	if v := os.Getenv("APIGATE_TLS_CERT"); v != "" {
		cfg.TLS.CertPath = v
	}
	if v := os.Getenv("APIGATE_TLS_KEY"); v != "" {
		cfg.TLS.KeyPath = v
	}
	if v := os.Getenv("APIGATE_TLS_HTTP_REDIRECT"); v != "" {
		cfg.TLS.HTTPRedirect = parseBool(v)
	}
	if v := os.Getenv("APIGATE_TLS_MIN_VERSION"); v != "" {
		cfg.TLS.MinVersion = v
	}
	if v := os.Getenv("APIGATE_TLS_ACME_STAGING"); v != "" {
		cfg.TLS.ACMEStaging = parseBool(v)
	}
}

// parseBool parses a boolean from common string values.
func parseBool(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "true" || v == "1" || v == "yes" || v == "on"
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

	// Portal defaults
	if cfg.Portal.AppName == "" {
		cfg.Portal.AppName = "APIGate"
	}

	// Email defaults
	if cfg.Email.Provider == "" {
		cfg.Email.Provider = "none"
	}
	if cfg.Email.SMTP.Port == 0 {
		cfg.Email.SMTP.Port = 587
	}
	if cfg.Email.SMTP.Timeout == 0 {
		cfg.Email.SMTP.Timeout = 30 * time.Second
	}
	if cfg.Email.SMTP.FromName == "" {
		cfg.Email.SMTP.FromName = cfg.Portal.AppName
	}

	// TLS defaults
	if cfg.TLS.Mode == "" {
		cfg.TLS.Mode = "none"
	}
	if cfg.TLS.MinVersion == "" {
		cfg.TLS.MinVersion = "1.2"
	}
	// HTTPRedirect defaults to true when TLS is enabled
	if cfg.TLS.Enabled && !cfg.TLS.HTTPRedirect {
		cfg.TLS.HTTPRedirect = true
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

	// TLS validation
	validTLSModes := map[string]bool{"none": true, "acme": true, "manual": true}
	if !validTLSModes[cfg.TLS.Mode] {
		return fmt.Errorf("tls.mode must be 'none', 'acme', or 'manual', got %q", cfg.TLS.Mode)
	}
	if cfg.TLS.Enabled {
		switch cfg.TLS.Mode {
		case "acme":
			if cfg.TLS.Domain == "" {
				return fmt.Errorf("tls.domain is required when tls.mode is 'acme'")
			}
		case "manual":
			if cfg.TLS.CertPath == "" || cfg.TLS.KeyPath == "" {
				return fmt.Errorf("tls.cert_path and tls.key_path are required when tls.mode is 'manual'")
			}
		case "none":
			return fmt.Errorf("tls.enabled is true but tls.mode is 'none'; set tls.mode to 'acme' or 'manual'")
		}
	}
	validTLSVersions := map[string]bool{"1.2": true, "1.3": true}
	if !validTLSVersions[cfg.TLS.MinVersion] {
		return fmt.Errorf("tls.min_version must be '1.2' or '1.3', got %q", cfg.TLS.MinVersion)
	}

	return nil
}
