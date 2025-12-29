package settings_test

import (
	"testing"
	"time"

	"github.com/artpar/apigate/domain/settings"
)

func TestSettings_Get(t *testing.T) {
	s := settings.Settings{
		"key1": "value1",
		"key2": "value2",
	}

	if s.Get("key1") != "value1" {
		t.Error("expected value1")
	}
	if s.Get("missing") != "" {
		t.Error("expected empty for missing key")
	}
}

func TestSettings_GetOrDefault(t *testing.T) {
	s := settings.Settings{
		"key1":  "value1",
		"empty": "",
	}

	if s.GetOrDefault("key1", "default") != "value1" {
		t.Error("expected value1")
	}
	if s.GetOrDefault("missing", "default") != "default" {
		t.Error("expected default for missing key")
	}
	if s.GetOrDefault("empty", "default") != "default" {
		t.Error("expected default for empty value")
	}
}

func TestSettings_GetBool(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			s := settings.Settings{"key": tt.value}
			if s.GetBool("key") != tt.want {
				t.Errorf("GetBool(%q) = %v, want %v", tt.value, s.GetBool("key"), tt.want)
			}
		})
	}
}

func TestSettings_GetInt(t *testing.T) {
	s := settings.Settings{
		"valid":   "42",
		"zero":    "0",
		"negative": "-10",
		"invalid": "not-a-number",
		"empty":   "",
	}

	if s.GetInt("valid", 0) != 42 {
		t.Error("expected 42")
	}
	if s.GetInt("zero", 99) != 0 {
		t.Error("expected 0")
	}
	if s.GetInt("negative", 0) != -10 {
		t.Error("expected -10")
	}
	if s.GetInt("invalid", 99) != 99 {
		t.Error("expected default for invalid")
	}
	if s.GetInt("empty", 99) != 99 {
		t.Error("expected default for empty")
	}
	if s.GetInt("missing", 99) != 99 {
		t.Error("expected default for missing")
	}
}

func TestSettings_GetDuration(t *testing.T) {
	s := settings.Settings{
		"valid":   "30s",
		"minutes": "5m",
		"hours":   "2h",
		"invalid": "not-duration",
		"empty":   "",
	}

	if s.GetDuration("valid", 0) != 30*time.Second {
		t.Error("expected 30s")
	}
	if s.GetDuration("minutes", 0) != 5*time.Minute {
		t.Error("expected 5m")
	}
	if s.GetDuration("hours", 0) != 2*time.Hour {
		t.Error("expected 2h")
	}
	if s.GetDuration("invalid", time.Minute) != time.Minute {
		t.Error("expected default for invalid")
	}
	if s.GetDuration("empty", time.Minute) != time.Minute {
		t.Error("expected default for empty")
	}
	if s.GetDuration("missing", time.Minute) != time.Minute {
		t.Error("expected default for missing")
	}
}

func TestSensitiveKeys(t *testing.T) {
	keys := settings.SensitiveKeys()
	if len(keys) == 0 {
		t.Error("expected sensitive keys")
	}

	// Check some known sensitive keys
	found := make(map[string]bool)
	for _, k := range keys {
		found[k] = true
	}

	expected := []string{
		settings.KeyAuthJWTSecret,
		settings.KeyEmailSMTPPassword,
		settings.KeyPaymentStripeSecretKey,
	}

	for _, k := range expected {
		if !found[k] {
			t.Errorf("expected %s to be sensitive", k)
		}
	}
}

func TestIsSensitive(t *testing.T) {
	if !settings.IsSensitive(settings.KeyAuthJWTSecret) {
		t.Error("JWT secret should be sensitive")
	}
	if !settings.IsSensitive(settings.KeyPaymentStripeSecretKey) {
		t.Error("Stripe secret should be sensitive")
	}
	if settings.IsSensitive(settings.KeyServerHost) {
		t.Error("Server host should not be sensitive")
	}
	if settings.IsSensitive("random.key") {
		t.Error("Random key should not be sensitive")
	}
}

func TestDefaults(t *testing.T) {
	d := settings.Defaults()

	if d.Get(settings.KeyServerPort) != "8080" {
		t.Errorf("expected default port 8080, got %s", d.Get(settings.KeyServerPort))
	}
	if d.Get(settings.KeyAuthKeyPrefix) != "ak_" {
		t.Error("expected default key prefix ak_")
	}
	if d.Get(settings.KeyRateLimitEnabled) != "true" {
		t.Error("expected rate limit enabled by default")
	}
	if d.Get(settings.KeyEmailProvider) != "none" {
		t.Error("expected email provider none by default")
	}
}

func TestMerge(t *testing.T) {
	loaded := settings.Settings{
		settings.KeyServerPort:     "9000",
		settings.KeyEmailProvider:  "smtp",
		"custom.key":               "custom-value",
	}

	merged := settings.Merge(loaded)

	// Loaded values should override defaults
	if merged.Get(settings.KeyServerPort) != "9000" {
		t.Error("expected loaded port 9000")
	}
	if merged.Get(settings.KeyEmailProvider) != "smtp" {
		t.Error("expected loaded email provider")
	}

	// Default values should be present
	if merged.Get(settings.KeyServerHost) != "0.0.0.0" {
		t.Error("expected default host")
	}
	if merged.Get(settings.KeyAuthKeyPrefix) != "ak_" {
		t.Error("expected default key prefix")
	}

	// Custom keys should be preserved
	if merged.Get("custom.key") != "custom-value" {
		t.Error("expected custom key")
	}
}

func TestSettingStruct(t *testing.T) {
	s := settings.Setting{
		Key:       "test.key",
		Value:     "test-value",
		Encrypted: true,
		UpdatedAt: time.Now(),
	}

	if s.Key != "test.key" {
		t.Error("expected key")
	}
	if s.Value != "test-value" {
		t.Error("expected value")
	}
	if !s.Encrypted {
		t.Error("expected encrypted")
	}
}

func TestConstants(t *testing.T) {
	// Verify key constants are defined
	keys := []string{
		settings.KeyServerHost,
		settings.KeyServerPort,
		settings.KeyPortalEnabled,
		settings.KeyEmailProvider,
		settings.KeyPaymentProvider,
		settings.KeyAuthMode,
		settings.KeyRateLimitEnabled,
		settings.KeyUpstreamURL,
	}

	for _, k := range keys {
		if k == "" {
			t.Errorf("key constant should not be empty")
		}
	}
}
