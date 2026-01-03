// Package settings provides value types for application settings.
// Settings are stored in the database and loaded at runtime.
package settings

import (
	"encoding/json"
	"time"
)

// Setting represents a single configuration setting (immutable value type).
type Setting struct {
	Key       string
	Value     string
	Encrypted bool
	UpdatedAt time.Time
}

// Settings is a collection of settings with helper methods.
type Settings map[string]string

// Get returns a setting value or empty string if not found.
func (s Settings) Get(key string) string {
	return s[key]
}

// GetOrDefault returns a setting value or the default if not found.
func (s Settings) GetOrDefault(key, defaultValue string) string {
	if v, ok := s[key]; ok && v != "" {
		return v
	}
	return defaultValue
}

// GetBool returns a setting as bool (true if "true", "1", "yes", "on").
func (s Settings) GetBool(key string) bool {
	v := s[key]
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

// GetInt returns a setting as int or default if not found/invalid.
func (s Settings) GetInt(key string, defaultValue int) int {
	v := s[key]
	if v == "" {
		return defaultValue
	}
	var i int
	if err := json.Unmarshal([]byte(v), &i); err != nil {
		return defaultValue
	}
	return i
}

// GetDuration returns a setting as duration or default if not found/invalid.
func (s Settings) GetDuration(key string, defaultValue time.Duration) time.Duration {
	v := s[key]
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}
	return d
}

// Known setting keys (namespaced by category).
const (
	// Server settings
	KeyServerHost         = "server.host"
	KeyServerPort         = "server.port"
	KeyServerReadTimeout  = "server.read_timeout"
	KeyServerWriteTimeout = "server.write_timeout"

	// Portal settings
	KeyPortalEnabled = "portal.enabled"
	KeyPortalBaseURL = "portal.base_url"
	KeyPortalAppName = "portal.app_name"

	// Email settings
	KeyEmailProvider     = "email.provider" // smtp, sendgrid, ses, postmark, none
	KeyEmailFromAddress  = "email.from_address"
	KeyEmailFromName     = "email.from_name"
	KeyEmailSMTPHost     = "email.smtp.host"
	KeyEmailSMTPPort     = "email.smtp.port"
	KeyEmailSMTPUsername = "email.smtp.username"
	KeyEmailSMTPPassword = "email.smtp.password"
	KeyEmailSMTPUseTLS   = "email.smtp.use_tls"
	KeyEmailSendGridKey  = "email.sendgrid.api_key"
	KeyEmailSESRegion    = "email.ses.region"
	KeyEmailSESAccessKey = "email.ses.access_key"
	KeyEmailSESSecretKey = "email.ses.secret_key"

	// Payment settings
	KeyPaymentProvider           = "payment.provider" // stripe, paddle, lemonsqueezy, none
	KeyPaymentStripeSecretKey    = "payment.stripe.secret_key"
	KeyPaymentStripePublicKey    = "payment.stripe.public_key"
	KeyPaymentStripeWebhookSecret = "payment.stripe.webhook_secret"
	KeyPaymentPaddleVendorID     = "payment.paddle.vendor_id"
	KeyPaymentPaddleAPIKey       = "payment.paddle.api_key"
	KeyPaymentPaddlePublicKey    = "payment.paddle.public_key"
	KeyPaymentPaddleWebhookSecret = "payment.paddle.webhook_secret"
	KeyPaymentLemonAPIKey        = "payment.lemonsqueezy.api_key"
	KeyPaymentLemonStoreID       = "payment.lemonsqueezy.store_id"
	KeyPaymentLemonWebhookSecret = "payment.lemonsqueezy.webhook_secret"

	// Auth settings
	KeyAuthMode                     = "auth.mode"
	KeyAuthHeader                   = "auth.header"
	KeyAuthJWTSecret                = "auth.jwt_secret"
	KeyAuthKeyPrefix                = "auth.key_prefix"
	KeyAuthSessionTTL               = "auth.session_ttl"
	KeyAuthRequireEmailVerification = "auth.require_email_verification"

	// Rate limit settings
	KeyRateLimitEnabled     = "ratelimit.enabled"
	KeyRateLimitBurstTokens = "ratelimit.burst_tokens"
	KeyRateLimitWindowSecs  = "ratelimit.window_secs"

	// Upstream settings (default upstream when no route matches)
	KeyUpstreamURL            = "upstream.url"
	KeyUpstreamTimeout        = "upstream.timeout"
	KeyUpstreamMaxIdleConns   = "upstream.max_idle_conns"
	KeyUpstreamIdleConnTimeout = "upstream.idle_conn_timeout"

	// Terminology settings (customize UI labels for different metering modes)
	KeyMeteringUnit = "metering.unit" // requests, tokens, data_points, bytes
)

// SensitiveKeys returns keys that contain secrets and should be encrypted.
func SensitiveKeys() []string {
	return []string{
		KeyAuthJWTSecret,
		KeyEmailSMTPPassword,
		KeyEmailSendGridKey,
		KeyEmailSESSecretKey,
		KeyPaymentStripeSecretKey,
		KeyPaymentStripeWebhookSecret,
		KeyPaymentPaddleAPIKey,
		KeyPaymentPaddleWebhookSecret,
		KeyPaymentLemonAPIKey,
		KeyPaymentLemonWebhookSecret,
	}
}

// IsSensitive returns true if the key contains sensitive data.
func IsSensitive(key string) bool {
	for _, k := range SensitiveKeys() {
		if k == key {
			return true
		}
	}
	return false
}

// Defaults returns default values for settings.
func Defaults() Settings {
	return Settings{
		KeyServerHost:                   "0.0.0.0",
		KeyServerPort:                   "8080",
		KeyServerReadTimeout:            "30s",
		KeyServerWriteTimeout:           "60s",
		KeyPortalEnabled:                "true",
		KeyPortalAppName:                "APIGate",
		KeyAuthRequireEmailVerification: "false",
		KeyEmailProvider:       "none",
		KeyPaymentProvider:     "none",
		KeyAuthMode:            "local",
		KeyAuthHeader:          "X-API-Key",
		KeyAuthKeyPrefix:       "ak_",
		KeyAuthSessionTTL:      "168h", // 7 days
		KeyRateLimitEnabled:    "true",
		KeyRateLimitBurstTokens: "5",
		KeyRateLimitWindowSecs:  "60",
		KeyUpstreamTimeout:      "30s",
		KeyUpstreamMaxIdleConns: "100",
		KeyUpstreamIdleConnTimeout: "90s",
		KeyMeteringUnit:         "requests",
	}
}

// Merge merges defaults with loaded settings, preferring loaded values.
func Merge(loaded Settings) Settings {
	result := Defaults()
	for k, v := range loaded {
		result[k] = v
	}
	return result
}
