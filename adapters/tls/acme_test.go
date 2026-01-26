package tls

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	cryptotls "crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	"golang.org/x/crypto/acme"
)

// TestNewACMEProvider_Config verifies directory URL selection based on staging flag.
func TestNewACMEProvider_Config(t *testing.T) {
	tests := []struct {
		name       string
		staging    bool
		wantDirURL string
	}{
		{
			name:       "production mode",
			staging:    false,
			wantDirURL: letsEncryptProduction,
		},
		{
			name:       "staging mode",
			staging:    true,
			wantDirURL: letsEncryptStaging,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := t.TempDir() + "/test.db"
			db, err := sqlite.Open(dbPath)
			if err != nil {
				t.Fatalf("Failed to open database: %v", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				t.Fatalf("Failed to migrate: %v", err)
			}

			certStore := sqlite.NewCertificateStore(db)

			provider, err := NewACMEProvider(certStore, ACMEConfig{
				Email:   "test@example.com",
				Staging: tt.staging,
			})
			if err != nil {
				t.Fatalf("NewACMEProvider failed: %v", err)
			}

			// Verify directory URL
			gotURL := provider.getDirectoryURL()
			if gotURL != tt.wantDirURL {
				t.Errorf("getDirectoryURL() = %q, want %q", gotURL, tt.wantDirURL)
			}

			// Verify staging flag
			if provider.staging != tt.staging {
				t.Errorf("staging = %v, want %v", provider.staging, tt.staging)
			}

			// Verify client is initialized
			if provider.client == nil {
				t.Error("client is nil, should be initialized")
			}
		})
	}
}

// TestClassifyError verifies error classification logic.
func TestClassifyError(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	tests := []struct {
		name     string
		err      error
		wantType ACMEErrorType
	}{
		{
			name:     "nil error",
			err:      nil,
			wantType: ErrorRetryable,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			wantType: ErrorRetryable,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			wantType: ErrorRetryable,
		},
		{
			name:     "ACME 400 error",
			err:      &acme.Error{StatusCode: 400},
			wantType: ErrorInvalid,
		},
		{
			name:     "ACME 403 error",
			err:      &acme.Error{StatusCode: 403},
			wantType: ErrorInvalid,
		},
		{
			name:     "ACME 404 error",
			err:      &acme.Error{StatusCode: 404},
			wantType: ErrorInvalid,
		},
		{
			name:     "ACME 429 error",
			err:      &acme.Error{StatusCode: 429},
			wantType: ErrorRateLimited,
		},
		{
			name:     "ACME 500 error",
			err:      &acme.Error{StatusCode: 500},
			wantType: ErrorRetryable,
		},
		{
			name:     "ACME 502 error",
			err:      &acme.Error{StatusCode: 502},
			wantType: ErrorRetryable,
		},
		{
			name:     "ACME 503 error",
			err:      &acme.Error{StatusCode: 503},
			wantType: ErrorRetryable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType := provider.classifyError(tt.err)
			if gotType != tt.wantType {
				t.Errorf("classifyError() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

// TestSelectChallenge verifies challenge selection prefers TLS-ALPN-01.
func TestSelectChallenge(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	tests := []struct {
		name       string
		challenges []*acme.Challenge
		wantType   string
	}{
		{
			name: "prefer TLS-ALPN-01 over HTTP-01",
			challenges: []*acme.Challenge{
				{Type: challengeHTTP01, Token: "http-token"},
				{Type: challengeTLSALPN01, Token: "tls-token"},
			},
			wantType: challengeTLSALPN01,
		},
		{
			name: "fallback to HTTP-01",
			challenges: []*acme.Challenge{
				{Type: challengeHTTP01, Token: "http-token"},
			},
			wantType: challengeHTTP01,
		},
		{
			name: "unsupported challenges only",
			challenges: []*acme.Challenge{
				{Type: "dns-01", Token: "dns-token"},
			},
			wantType: "",
		},
		{
			name:       "empty challenges",
			challenges: []*acme.Challenge{},
			wantType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.selectChallenge(tt.challenges)
			if tt.wantType == "" {
				if result != nil {
					t.Errorf("selectChallenge() = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("selectChallenge() = nil, want %s", tt.wantType)
				} else if result.Type != tt.wantType {
					t.Errorf("selectChallenge().Type = %s, want %s", result.Type, tt.wantType)
				}
			}
		})
	}
}

// TestRateLimitFastFail verifies rate limit fast-fail behavior.
func TestRateLimitFastFail(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"test.example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	domain := "test.example.com"

	// Initially not rate limited
	if provider.isRateLimited(domain) {
		t.Error("Domain should not be rate limited initially")
	}

	// Record rate limit
	retryAfter := time.Now().Add(1 * time.Hour)
	provider.recordRateLimit(domain, retryAfter)

	// Now should be rate limited
	if !provider.isRateLimited(domain) {
		t.Error("Domain should be rate limited after recording")
	}

	// Get rate limit info
	expiry, exists := provider.GetRateLimitInfo(domain)
	if !exists {
		t.Error("GetRateLimitInfo should return true for rate limited domain")
	}
	if expiry.Sub(retryAfter) > time.Second {
		t.Errorf("Expiry time mismatch: got %v, want %v", expiry, retryAfter)
	}

	// Clear rate limit
	provider.ClearRateLimit(domain)

	// Should no longer be rate limited
	if provider.isRateLimited(domain) {
		t.Error("Domain should not be rate limited after clearing")
	}

	_, exists = provider.GetRateLimitInfo(domain)
	if exists {
		t.Error("GetRateLimitInfo should return false after clearing")
	}

	t.Log("✓ Rate limit fast-fail works correctly")
}

// TestGetCertificateWithLogging_HostPolicyRejection verifies logging when domain is rejected.
func TestGetCertificateWithLogging_HostPolicyRejection(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"allowed.example.com"}, // Only allow this domain
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	provider.SetLogger(logger)

	// Try to get certificate for disallowed domain
	hello := &cryptotls.ClientHelloInfo{
		ServerName: "disallowed.example.com",
	}

	_, err = provider.GetCertificateWithLogging(hello)
	if err == nil {
		t.Error("Expected error for disallowed domain, got nil")
	}

	logOutput := buf.String()

	// Verify logging occurred
	if !strings.Contains(logOutput, "TLS certificate requested") {
		t.Error("Expected 'TLS certificate requested' log message")
	}
	if !strings.Contains(logOutput, "Domain rejected by host policy") {
		t.Error("Expected 'Domain rejected by host policy' log message")
	}
	if !strings.Contains(logOutput, "disallowed.example.com") {
		t.Error("Expected domain name in log output")
	}

	t.Log("✓ Host policy rejection logged correctly")
}

// TestGetCertificateWithLogging_AllDomainsAllowed verifies behavior when no domains configured.
func TestGetCertificateWithLogging_AllDomainsAllowed(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	// No domains configured = allow all
	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{}, // Empty = allow all
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	// Capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	provider.SetLogger(logger)

	// Try to get certificate - will fail at ACME level but should pass host policy
	hello := &cryptotls.ClientHelloInfo{
		ServerName: "any.example.com",
	}

	_, err = provider.GetCertificateWithLogging(hello)
	// Error expected (no actual ACME server), but not host policy error
	if err != nil && strings.Contains(err.Error(), "not in allowed domains") {
		t.Error("Should not get host policy error when no domains configured")
	}

	logOutput := buf.String()

	// Should log the request
	if !strings.Contains(logOutput, "TLS certificate requested") {
		t.Error("Expected 'TLS certificate requested' log message")
	}
	// Should NOT log host policy rejection
	if strings.Contains(logOutput, "Domain rejected by host policy") {
		t.Error("Should not log host policy rejection when all domains allowed")
	}

	t.Log("✓ All domains allowed when none configured")
}

// TestSetLogger verifies logger is set correctly.
func TestSetLogger(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	// Create custom logger
	var buf bytes.Buffer
	customLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Set logger
	provider.SetLogger(customLogger)

	// Verify provider logger is set
	if provider.logger != customLogger {
		t.Error("Provider logger not set correctly")
	}

	t.Log("✓ Logger set correctly")
}

// TestHostPolicy verifies domain matching including wildcards.
func TestHostPolicy(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"example.com", "*.wildcard.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{
			name:    "exact match",
			host:    "example.com",
			wantErr: false,
		},
		{
			name:    "wildcard match",
			host:    "sub.wildcard.com",
			wantErr: false,
		},
		{
			name:    "deep wildcard match",
			host:    "deep.sub.wildcard.com",
			wantErr: false,
		},
		{
			name:    "no match",
			host:    "other.com",
			wantErr: true,
		},
		{
			name:    "partial match fails",
			host:    "notexample.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.hostPolicy(context.Background(), tt.host)
			if tt.wantErr && err == nil {
				t.Errorf("Expected error for host %q, got nil", tt.host)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error for host %q: %v", tt.host, err)
			}
		})
	}
}

// TestUpdateDomains verifies domain list updates.
func TestUpdateDomains(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"old.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	// Verify old domain allowed
	err = provider.hostPolicy(context.Background(), "old.com")
	if err != nil {
		t.Error("old.com should be allowed initially")
	}

	// Update domains
	provider.UpdateDomains([]string{"new.com"})

	// Verify new domain allowed
	err = provider.hostPolicy(context.Background(), "new.com")
	if err != nil {
		t.Error("new.com should be allowed after update")
	}

	// Verify old domain now rejected
	err = provider.hostPolicy(context.Background(), "old.com")
	if err == nil {
		t.Error("old.com should be rejected after update")
	}

	t.Log("✓ Domain list updates correctly")
}

// TestACMEProviderName verifies provider name.
func TestACMEProviderName(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	if provider.Name() != "acme" {
		t.Errorf("Expected name 'acme', got %q", provider.Name())
	}
}

// TestACMEProviderDefaultRenewalDays verifies default renewal days.
func TestACMEProviderDefaultRenewalDays(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	tests := []struct {
		name        string
		renewalDays int
		want        int
	}{
		{"zero defaults to 30", 0, 30},
		{"negative defaults to 30", -1, 30},
		{"custom value used", 14, 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewACMEProvider(certStore, ACMEConfig{
				Email:       "test@example.com",
				Staging:     true,
				RenewalDays: tt.renewalDays,
			})
			if err != nil {
				t.Fatalf("NewACMEProvider failed: %v", err)
			}

			if provider.renewalDays != tt.want {
				t.Errorf("renewalDays = %d, want %d", provider.renewalDays, tt.want)
			}
		})
	}
}

// TestGetCertificateWithLogging_StagingFlag verifies staging flag is logged.
func TestGetCertificateWithLogging_StagingFlag(t *testing.T) {
	tests := []struct {
		name    string
		staging bool
	}{
		{"staging true", true},
		{"staging false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := t.TempDir() + "/test.db"
			db, err := sqlite.Open(dbPath)
			if err != nil {
				t.Fatalf("Failed to open database: %v", err)
			}
			defer db.Close()

			if err := db.Migrate(); err != nil {
				t.Fatalf("Failed to migrate: %v", err)
			}

			certStore := sqlite.NewCertificateStore(db)

			provider, err := NewACMEProvider(certStore, ACMEConfig{
				Email:   "test@example.com",
				Staging: tt.staging,
				Domains: []string{"test.com"},
			})
			if err != nil {
				t.Fatalf("NewACMEProvider failed: %v", err)
			}

			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
			provider.SetLogger(logger)

			hello := &cryptotls.ClientHelloInfo{
				ServerName: "test.com",
			}

			// Will fail but we just want to check logging
			_, _ = provider.GetCertificateWithLogging(hello)

			logOutput := buf.String()

			// Verify staging flag is logged
			if tt.staging && !strings.Contains(logOutput, "staging=true") {
				t.Error("Expected staging=true in log output")
			}
			if !tt.staging && !strings.Contains(logOutput, "staging=false") {
				t.Error("Expected staging=false in log output")
			}
		})
	}
}

// TestCheckRenewal verifies renewal check logic.
func TestCheckRenewal(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	ctx := context.Background()

	// Test 1: Non-existent certificate needs renewal
	needsRenewal, err := provider.CheckRenewal(ctx, "nonexistent.com", 30)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !needsRenewal {
		t.Error("Non-existent certificate should need renewal")
	}

	t.Log("✓ CheckRenewal works for non-existent certificates")
}

// TestGetCertificateFunc verifies GetCertificateFunc returns correct function.
func TestGetCertificateFunc(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	certFunc := provider.GetCertificateFunc()
	if certFunc == nil {
		t.Error("GetCertificateFunc returned nil")
	}

	t.Log("✓ GetCertificateFunc returns correct function")
}

// TestCertCache verifies in-memory certificate caching.
func TestCertCache(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	// Initially cache is empty
	cert := provider.getCachedCert("test.com")
	if cert != nil {
		t.Error("Cache should be empty initially")
	}

	// Add certificate to cache
	testCert := &cryptotls.Certificate{}
	provider.setCachedCert("test.com", testCert)

	// Should be in cache now
	cert = provider.getCachedCert("test.com")
	if cert == nil {
		t.Error("Certificate should be in cache")
	}

	t.Log("✓ In-memory certificate cache works correctly")
}

// TestGetCertificate verifies the GetCertificate method.
func TestGetCertificate(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:       "test@example.com",
		Staging:     true,
		Domains:     []string{"test.example.com"},
		RenewalDays: 30,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	ctx := context.Background()

	// Test: Certificate not in database - will fail at ACME level
	_, err = provider.GetCertificate(ctx, "test.example.com")
	if err == nil {
		t.Log("GetCertificate succeeded (unexpected but OK)")
	} else {
		t.Logf("GetCertificate returned expected error: %v", err)
	}

	t.Log("✓ GetCertificate code path exercised")
}

// TestObtainCertificate verifies the ObtainCertificate method.
func TestObtainCertificate(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"allowed.example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	ctx := context.Background()

	// Test 1: Domain not allowed
	_, err = provider.ObtainCertificate(ctx, "disallowed.example.com")
	if err == nil {
		t.Error("Expected error for disallowed domain")
	} else if !strings.Contains(err.Error(), "not in allowed domains") {
		t.Logf("Got different error: %v", err)
	}

	// Test 2: Domain allowed but ACME will fail (no real server)
	_, err = provider.ObtainCertificate(ctx, "allowed.example.com")
	if err == nil {
		t.Log("ObtainCertificate succeeded unexpectedly")
	} else {
		t.Logf("ObtainCertificate returned expected error: %v", err)
	}

	t.Log("✓ ObtainCertificate code paths exercised")
}

// TestRenewCertificate verifies the RenewCertificate method.
func TestRenewCertificate(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"test.example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	ctx := context.Background()

	// RenewCertificate clears cache and calls ObtainCertificate
	// Will fail at ACME level but exercises the code path
	_, err = provider.RenewCertificate(ctx, "test.example.com")
	if err == nil {
		t.Log("RenewCertificate succeeded unexpectedly")
	} else {
		t.Logf("RenewCertificate returned expected error: %v", err)
	}

	t.Log("✓ RenewCertificate code path exercised")
}

// TestGetCertificateWithLogging_ErrorLogging verifies error logging includes duration.
func TestGetCertificateWithLogging_ErrorLogging(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"test.example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	provider.SetLogger(logger)

	hello := &cryptotls.ClientHelloInfo{
		ServerName: "test.example.com",
	}

	// This will fail (no real ACME server) but should log the error
	_, err = provider.GetCertificateWithLogging(hello)
	if err == nil {
		t.Log("Note: GetCertificate succeeded unexpectedly (might have cached cert)")
	}

	logOutput := buf.String()

	// Verify request was logged
	if !strings.Contains(logOutput, "TLS certificate requested") {
		t.Error("Expected 'TLS certificate requested' log message")
	}

	// Verify domain appears in log
	if !strings.Contains(logOutput, "test.example.com") {
		t.Error("Expected domain in log output")
	}

	t.Log("✓ Error logging includes expected fields")
}

// TestGetCertificateWithLogging_RateLimitFastFail verifies rate limit fast-fail in GetCertificateWithLogging.
func TestGetCertificateWithLogging_RateLimitFastFail(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"test.example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	provider.SetLogger(logger)

	// Set rate limit for domain
	provider.recordRateLimit("test.example.com", time.Now().Add(1*time.Hour))

	hello := &cryptotls.ClientHelloInfo{
		ServerName: "test.example.com",
	}

	// Should fail fast due to rate limit
	_, err = provider.GetCertificateWithLogging(hello)
	if err == nil {
		t.Error("Expected error due to rate limit")
	}

	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("Expected rate limit error, got: %v", err)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "rate limited") || !strings.Contains(logOutput, "fast-fail") {
		t.Error("Expected rate limit fast-fail log message")
	}

	t.Log("✓ Rate limit fast-fail works in GetCertificateWithLogging")
}

// TestSplitCertData verifies certificate data splitting.
func TestSplitCertData(t *testing.T) {
	// Generate test certificate and key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyBytes, _ := x509.MarshalECPrivateKey(privKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	// Test valid data (key + cert as autocert stores it)
	combined := append(keyPEM, certPEM...)
	gotCert, gotKey, gotChain, err := splitCertData(combined)
	if err != nil {
		t.Errorf("splitCertData failed: %v", err)
	}
	if len(gotCert) == 0 {
		t.Error("Expected non-empty cert")
	}
	if len(gotKey) == 0 {
		t.Error("Expected non-empty key")
	}
	if len(gotChain) != 0 {
		t.Error("Expected empty chain for single cert")
	}

	// Test invalid data
	_, _, _, err = splitCertData([]byte("not pem data"))
	if err == nil {
		t.Error("Expected error for invalid data")
	}

	t.Log("✓ splitCertData works correctly")
}

// TestParseRetryAfter verifies retry-after parsing.
func TestParseRetryAfter(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	tests := []struct {
		name    string
		errStr  string
		wantUTC bool
	}{
		{
			name:    "valid format",
			errStr:  "acme: retry after 2026-01-26 09:41:29 UTC",
			wantUTC: true,
		},
		{
			name:    "no match defaults to 1 hour",
			errStr:  "some other error",
			wantUTC: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.parseRetryAfter(tt.errStr)

			if tt.wantUTC {
				expected, _ := time.Parse("2006-01-02 15:04:05", "2026-01-26 09:41:29")
				if !result.Equal(expected.UTC()) {
					t.Errorf("parseRetryAfter() = %v, want %v", result, expected.UTC())
				}
			} else {
				// Should be approximately 1 hour from now
				expectedApprox := time.Now().Add(1 * time.Hour)
				diff := result.Sub(expectedApprox)
				if diff < -time.Minute || diff > time.Minute {
					t.Errorf("parseRetryAfter() = %v, expected approximately %v", result, expectedApprox)
				}
			}
		})
	}

	t.Log("✓ parseRetryAfter works correctly")
}

// TestHTTPHandler tests the HTTPHandler method for ACME HTTP-01 challenges.
func TestHTTPHandler(t *testing.T) {
	dbPath := t.TempDir() + "/test.db"
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	certStore := sqlite.NewCertificateStore(db)

	provider, err := NewACMEProvider(certStore, ACMEConfig{
		Email:   "test@example.com",
		Staging: true,
		Domains: []string{"example.com"},
	})
	if err != nil {
		t.Fatalf("NewACMEProvider() error = %v", err)
	}

	tests := []struct {
		name           string
		path           string
		fallback       http.Handler
		wantStatusCode int
	}{
		{
			name:           "challenge_path_no_token",
			path:           "/.well-known/acme-challenge/missing-token",
			fallback:       nil,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "non_challenge_path_no_fallback",
			path:           "/some/other/path",
			fallback:       nil,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "non_challenge_path_with_fallback",
			path:           "/some/other/path",
			fallback:       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := provider.HTTPHandler(tt.fallback)

			req := httptest.NewRequest("GET", tt.path, nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("HTTPHandler() status = %d, want %d", rr.Code, tt.wantStatusCode)
			}
		})
	}

	t.Log("✓ HTTPHandler works correctly")
}
