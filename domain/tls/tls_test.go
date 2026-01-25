package tls

import (
	"strings"
	"testing"
	"time"
)

func TestMode_IsValid(t *testing.T) {
	tests := []struct {
		mode Mode
		want bool
	}{
		{ModeNone, true},
		{ModeACME, true},
		{ModeManual, true},
		{Mode("invalid"), false},
		{Mode(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got := tt.mode.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusActive, true},
		{StatusExpired, true},
		{StatusRevoked, true},
		{Status("invalid"), false},
		{Status(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateCertificateID(t *testing.T) {
	id := GenerateCertificateID()
	if !strings.HasPrefix(id, "cert_") {
		t.Errorf("GenerateCertificateID() = %v, want prefix cert_", id)
	}
	if len(id) != 21 { // cert_ (5) + 16 hex chars
		t.Errorf("GenerateCertificateID() length = %d, want 21", len(id))
	}

	// Ensure unique
	id2 := GenerateCertificateID()
	if id == id2 {
		t.Error("GenerateCertificateID() should generate unique IDs")
	}
}

func TestCertificate_WithID(t *testing.T) {
	c := Certificate{Domain: "example.com"}
	c2 := c.WithID("cert_123")

	if c2.ID != "cert_123" {
		t.Errorf("WithID() ID = %v, want cert_123", c2.ID)
	}
	if c2.Domain != "example.com" {
		t.Error("WithID() should preserve Domain")
	}
	if c.ID != "" {
		t.Error("WithID() should not modify original")
	}
}

func TestCertificate_WithStatus(t *testing.T) {
	c := Certificate{Domain: "example.com"}
	c2 := c.WithStatus(StatusRevoked)

	if c2.Status != StatusRevoked {
		t.Errorf("WithStatus() Status = %v, want revoked", c2.Status)
	}
	if c.Status != "" {
		t.Error("WithStatus() should not modify original")
	}
}

func TestCertificate_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"expired yesterday", time.Now().UTC().Add(-24 * time.Hour), true},
		{"expires tomorrow", time.Now().UTC().Add(24 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Certificate{ExpiresAt: tt.expiresAt}
			got := c.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificate_IsActive(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(30 * 24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tests := []struct {
		name   string
		cert   Certificate
		want   bool
	}{
		{"active and not expired", Certificate{Status: StatusActive, ExpiresAt: future}, true},
		{"active but expired", Certificate{Status: StatusActive, ExpiresAt: past}, false},
		{"revoked", Certificate{Status: StatusRevoked, ExpiresAt: future}, false},
		{"expired status", Certificate{Status: StatusExpired, ExpiresAt: future}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cert.IsActive()
			if got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificate_DaysUntilExpiry(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      int
	}{
		{"30 days from now", time.Now().UTC().Add(30 * 24 * time.Hour), 30},
		{"expired yesterday", time.Now().UTC().Add(-24 * time.Hour), -1},
		{"expires today", time.Now().UTC(), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Certificate{ExpiresAt: tt.expiresAt}
			got := c.DaysUntilExpiry()
			// Allow 1 day tolerance for test timing
			if got < tt.want-1 || got > tt.want+1 {
				t.Errorf("DaysUntilExpiry() = %v, want approximately %v", got, tt.want)
			}
		})
	}
}

func TestCertificate_NeedsRenewal(t *testing.T) {
	tests := []struct {
		name        string
		expiresAt   time.Time
		renewalDays int
		want        bool
	}{
		{"needs renewal", time.Now().UTC().Add(15 * 24 * time.Hour), 30, true},
		{"no renewal needed", time.Now().UTC().Add(60 * 24 * time.Hour), 30, false},
		{"expired already", time.Now().UTC().Add(-24 * time.Hour), 30, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Certificate{ExpiresAt: tt.expiresAt}
			got := c.NeedsRenewal(tt.renewalDays)
			if got != tt.want {
				t.Errorf("NeedsRenewal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificate_FullChain(t *testing.T) {
	t.Run("cert only", func(t *testing.T) {
		c := Certificate{CertPEM: []byte("CERT")}
		got := c.FullChain()
		if string(got) != "CERT" {
			t.Errorf("FullChain() = %v, want CERT", string(got))
		}
	})

	t.Run("cert with chain", func(t *testing.T) {
		c := Certificate{CertPEM: []byte("CERT"), ChainPEM: []byte("CHAIN")}
		got := c.FullChain()
		if string(got) != "CERT\nCHAIN" {
			t.Errorf("FullChain() = %v, want CERT\\nCHAIN", string(got))
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantValid  bool
		wantErrors []string
	}{
		{
			name:      "disabled is always valid",
			cfg:       Config{Enabled: false},
			wantValid: true,
		},
		{
			name:      "valid ACME config",
			cfg:       Config{Enabled: true, Mode: ModeACME, Domain: "example.com", Email: "admin@example.com"},
			wantValid: true,
		},
		{
			name:       "ACME missing domain",
			cfg:        Config{Enabled: true, Mode: ModeACME, Email: "admin@example.com"},
			wantValid:  false,
			wantErrors: []string{"domain"},
		},
		{
			name:       "ACME invalid domain",
			cfg:        Config{Enabled: true, Mode: ModeACME, Domain: "not a domain", Email: "admin@example.com"},
			wantValid:  false,
			wantErrors: []string{"domain"},
		},
		{
			name:       "ACME missing email",
			cfg:        Config{Enabled: true, Mode: ModeACME, Domain: "example.com"},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name:       "ACME invalid email",
			cfg:        Config{Enabled: true, Mode: ModeACME, Domain: "example.com", Email: "invalid"},
			wantValid:  false,
			wantErrors: []string{"email"},
		},
		{
			name:      "valid manual config",
			cfg:       Config{Enabled: true, Mode: ModeManual, CertPath: "/path/cert.pem", KeyPath: "/path/key.pem"},
			wantValid: true,
		},
		{
			name:       "manual missing cert path",
			cfg:        Config{Enabled: true, Mode: ModeManual, KeyPath: "/path/key.pem"},
			wantValid:  false,
			wantErrors: []string{"cert_path"},
		},
		{
			name:       "manual missing key path",
			cfg:        Config{Enabled: true, Mode: ModeManual, CertPath: "/path/cert.pem"},
			wantValid:  false,
			wantErrors: []string{"key_path"},
		},
		{
			name:       "invalid mode",
			cfg:        Config{Enabled: true, Mode: Mode("invalid")},
			wantValid:  false,
			wantErrors: []string{"mode"},
		},
		{
			name:       "invalid min version",
			cfg:        Config{Enabled: true, Mode: ModeNone, MinVersion: "1.0"},
			wantValid:  false,
			wantErrors: []string{"min_version"},
		},
		{
			name:      "valid min version 1.2",
			cfg:       Config{Enabled: true, Mode: ModeNone, MinVersion: "1.2"},
			wantValid: true,
		},
		{
			name:      "valid min version 1.3",
			cfg:       Config{Enabled: true, Mode: ModeNone, MinVersion: "1.3"},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateConfig(tt.cfg)
			if result.Valid != tt.wantValid {
				t.Errorf("ValidateConfig() Valid = %v, want %v, errors: %v", result.Valid, tt.wantValid, result.Errors)
			}
			for _, errKey := range tt.wantErrors {
				if _, ok := result.Errors[errKey]; !ok {
					t.Errorf("ValidateConfig() missing error for %v", errKey)
				}
			}
		})
	}
}

func TestACMEChallenge_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		expires time.Time
		want    bool
	}{
		{"expired", time.Now().UTC().Add(-time.Hour), true},
		{"not expired", time.Now().UTC().Add(time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ACMEChallenge{Expires: tt.expires}
			got := c.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestACMEChallenge_Path(t *testing.T) {
	c := ACMEChallenge{Token: "test-token-123"}
	got := c.Path()
	want := "/.well-known/acme-challenge/test-token-123"
	if got != want {
		t.Errorf("Path() = %v, want %v", got, want)
	}
}

func TestIsValidDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   bool
	}{
		{"example.com", true},
		{"sub.example.com", true},
		{"a.b.c.example.com", true},
		{"example123.com", true},
		{"example-test.com", true},
		{"localhost", false},
		{"example", false},
		{"", false},
		{"example.c", false},    // TLD too short
		{".example.com", false}, // leading dot
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := isValidDomain(tt.domain)
			if got != tt.want {
				t.Errorf("isValidDomain(%q) = %v, want %v", tt.domain, got, tt.want)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		want  bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := isValidEmail(tt.email)
			if got != tt.want {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestMinVersionToUint16(t *testing.T) {
	tests := []struct {
		version string
		want    uint16
	}{
		{"1.2", 0x0303},
		{"1.3", 0x0304},
		{"1.0", 0},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := MinVersionToUint16(tt.version)
			if got != tt.want {
				t.Errorf("MinVersionToUint16(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
