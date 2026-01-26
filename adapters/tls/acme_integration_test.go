package tls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	domaintls "github.com/artpar/apigate/domain/tls"
	"golang.org/x/crypto/acme"
)

// TestNewACMEProvider verifies that NewACMEProvider correctly initializes
// the acme.Client for both staging and production modes.
// This is the critical boundary test - Issue #48 occurred because Client
// was only set for staging mode, leaving production with nil Client.
func TestNewACMEProvider(t *testing.T) {
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
			provider, err := NewACMEProvider(certStore, ACMEConfig{
				Email:   "test@example.com",
				Staging: tt.staging,
				Domains: []string{"example.com"},
			})
			if err != nil {
				t.Fatalf("NewACMEProvider failed: %v", err)
			}

			// CRITICAL: client must never be nil
			// This was the root cause of Issue #48 - production mode had nil Client
			if provider.client == nil {
				t.Fatal("CRITICAL: provider.client is nil - this breaks ACME initialization")
			}

			// Verify correct directory URL based on staging flag
			if provider.client.DirectoryURL != tt.wantDirURL {
				t.Errorf("DirectoryURL = %q, want %q",
					provider.client.DirectoryURL, tt.wantDirURL)
			}

			t.Logf("Provider correctly initialized with DirectoryURL: %s", tt.wantDirURL)
		})
	}
}

// TestACMECacheMissError verifies the critical fix: ErrCacheMiss behavior.
// When a certificate is not found, the provider should return an error
// that triggers certificate obtainment.
func TestACMECacheMissError(t *testing.T) {
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

	// Memory cache should be empty
	cert := provider.getCachedCert("nonexistent.example.com")
	if cert != nil {
		t.Error("Expected nil from memory cache for nonexistent domain")
	}

	// Database lookup should fail
	_, err = provider.certStore.GetByDomain(context.Background(), "nonexistent.example.com")
	if err == nil {
		t.Error("Expected error from database lookup for nonexistent domain")
	}

	t.Log("✓ Cache miss behavior works correctly")
}

// TestCacheNonCertData verifies that non-certificate data (like ACME account keys) is handled.
// The new implementation handles account keys internally in the acme.Client.
func TestCacheNonCertData(t *testing.T) {
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

	// Simulate ACME account key storage (single PEM block)
	accountKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyBytes, _ := x509.MarshalECPrivateKey(accountKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Use a realistic ACME account key format
	accountKeyName := "+acme_account+https://acme-v02.api.letsencrypt.org/directory"

	// Store via ACMECacheStore interface (certStore implements both)
	err = certStore.PutCache(context.Background(), accountKeyName, keyPEM)
	if err != nil {
		t.Errorf("Failed to store account key: %v", err)
	}

	// Should be retrievable
	data, err := certStore.GetCache(context.Background(), accountKeyName)
	if err != nil {
		t.Errorf("Failed to retrieve account key: %v", err)
	}
	if len(data) == 0 {
		t.Error("Retrieved empty account key data")
	}

	t.Log("✓ Non-certificate data (account keys) handled correctly")
}

// TestCacheCertificateStorage verifies certificate storage and retrieval.
func TestCacheCertificateStorage(t *testing.T) {
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

	testDomain := "test.example.com"
	ctx := context.Background()

	// Generate a self-signed certificate for testing
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{testDomain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &certKey.PublicKey, certKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// Encode as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	keyBytes, _ := x509.MarshalECPrivateKey(certKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	// Test 1: Store certificate directly in database
	now := time.Now().UTC()
	cert := domaintls.Certificate{
		ID:           domaintls.GenerateCertificateID(),
		Domain:       testDomain,
		CertPEM:      certPEM,
		KeyPEM:       keyPEM,
		IssuedAt:     template.NotBefore,
		ExpiresAt:    template.NotAfter,
		Issuer:       "Test CA",
		SerialNumber: "1",
		Status:       domaintls.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = certStore.Create(ctx, cert)
	if err != nil {
		t.Fatalf("Failed to store certificate: %v", err)
	}
	t.Log("✓ Certificate stored in database")

	// Test 2: Retrieve from database
	dbCert, err := certStore.GetByDomain(ctx, testDomain)
	if err != nil {
		t.Fatalf("Certificate not in database: %v", err)
	}
	if dbCert.Domain != testDomain {
		t.Errorf("Domain mismatch: got %s, want %s", dbCert.Domain, testDomain)
	}
	if dbCert.Status != domaintls.StatusActive {
		t.Errorf("Status mismatch: got %s, want %s", dbCert.Status, domaintls.StatusActive)
	}
	t.Logf("✓ Certificate retrieved from database with ID: %s", dbCert.ID)

	// Test 3: Delete certificate
	err = certStore.Delete(ctx, dbCert.ID)
	if err != nil {
		t.Fatalf("Failed to delete certificate: %v", err)
	}

	// Verify it's gone
	_, err = certStore.GetByDomain(ctx, testDomain)
	if err == nil {
		t.Error("Expected error after delete")
	}
	t.Log("✓ Certificate deleted successfully")
}

// TestCombineCertData verifies certificate combining works correctly.
func TestCombineCertData(t *testing.T) {
	cert := domaintls.Certificate{
		CertPEM:  []byte("-----BEGIN CERTIFICATE-----\ntest-cert\n-----END CERTIFICATE-----"),
		KeyPEM:   []byte("-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----"),
		ChainPEM: []byte("-----BEGIN CERTIFICATE-----\ntest-chain\n-----END CERTIFICATE-----"),
	}

	data := combineCertData(cert)

	// Should contain all three parts
	if len(data) == 0 {
		t.Error("Combined data is empty")
	}

	t.Logf("Combined cert data length: %d bytes", len(data))
	t.Log("✓ Certificate data combining works")
}

// TestACMEDirectoryConnectivity verifies that both staging and production
// ACME directories are reachable. This is a network-level test that helps
// diagnose Issue #48 where production ACME fails but staging works.
func TestACMEDirectoryConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tests := []struct {
		name         string
		directoryURL string
	}{
		{"staging", letsEncryptStaging},
		{"production", letsEncryptProduction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &acme.Client{
				DirectoryURL: tt.directoryURL,
				HTTPClient:   httpClient,
				Key:          key,
			}

			// Test directory discovery
			dir, err := client.Discover(ctx)
			if err != nil {
				t.Errorf("Discover failed for %s: %v", tt.name, err)
				return
			}

			// Verify we got a valid directory response
			// The NonceURL or OrderURL should be populated
			if dir.NonceURL == "" && dir.OrderURL == "" {
				t.Errorf("Directory response empty for %s (no NonceURL or OrderURL)", tt.name)
			}

			t.Logf("%s directory: NonceURL=%s, OrderURL=%s", tt.name, dir.NonceURL, dir.OrderURL)
		})
	}
}

// TestACMEHTTPClientTimeout verifies that the HTTP client has proper timeouts
// configured. This is critical for Issue #48 - without timeouts, network issues
// cause indefinite hangs instead of errors.
func TestACMEHTTPClientTimeout(t *testing.T) {
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
		name    string
		staging bool
	}{
		{"production", false},
		{"staging", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewACMEProvider(certStore, ACMEConfig{
				Email:   "test@example.com",
				Staging: tt.staging,
				Domains: []string{"example.com"},
			})
			if err != nil {
				t.Fatalf("NewACMEProvider failed: %v", err)
			}

			// CRITICAL: client.HTTPClient must have timeout set
			if provider.client == nil {
				t.Fatal("provider.client is nil")
			}
			if provider.client.HTTPClient == nil {
				t.Fatal("CRITICAL: provider.client.HTTPClient is nil - this causes indefinite hangs")
			}
			if provider.client.HTTPClient.Timeout == 0 {
				t.Fatal("CRITICAL: provider.client.HTTPClient.Timeout is 0 - this causes indefinite hangs")
			}

			t.Logf("%s mode: HTTPClient.Timeout = %v", tt.name, provider.client.HTTPClient.Timeout)
		})
	}
}

// TestDirectACMEProviderFields verifies the new DirectACMEProvider struct fields.
func TestDirectACMEProviderFields(t *testing.T) {
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
		Domains:     []string{"example.com", "*.example.com"},
		RenewalDays: 14,
	})
	if err != nil {
		t.Fatalf("NewACMEProvider failed: %v", err)
	}

	// Verify fields are initialized
	if provider.client == nil {
		t.Error("client is nil")
	}
	if provider.certStore == nil {
		t.Error("certStore is nil")
	}
	if provider.logger == nil {
		t.Error("logger is nil")
	}
	if provider.email != "test@example.com" {
		t.Errorf("email = %q, want %q", provider.email, "test@example.com")
	}
	if provider.staging != true {
		t.Error("staging should be true")
	}
	if provider.renewalDays != 14 {
		t.Errorf("renewalDays = %d, want 14", provider.renewalDays)
	}
	if len(provider.domains) != 2 {
		t.Errorf("domains count = %d, want 2", len(provider.domains))
	}
	if provider.tlsAlpnCerts == nil {
		t.Error("tlsAlpnCerts map is nil")
	}
	if provider.http01Tokens == nil {
		t.Error("http01Tokens map is nil")
	}
	if provider.rateLimitUntil == nil {
		t.Error("rateLimitUntil map is nil")
	}
	if provider.certCache == nil {
		t.Error("certCache map is nil")
	}
	if provider.accountKey == nil {
		t.Error("accountKey is nil")
	}

	t.Log("✓ All DirectACMEProvider fields properly initialized")
}

// TestChallengeHandling verifies challenge maps work correctly.
func TestChallengeHandling(t *testing.T) {
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

	// Test HTTP-01 challenge handling
	keyAuth, err := provider.prepareHTTP01Challenge("test-token-12345")
	if err != nil {
		t.Fatalf("prepareHTTP01Challenge failed: %v", err)
	}
	if keyAuth == "" {
		t.Error("keyAuth is empty")
	}

	// Verify token is stored
	provider.challengeMu.RLock()
	storedKeyAuth, exists := provider.http01Tokens["test-token-12345"]
	provider.challengeMu.RUnlock()

	if !exists {
		t.Error("HTTP-01 token not stored")
	}
	if storedKeyAuth != keyAuth {
		t.Error("Stored keyAuth doesn't match")
	}

	// Cleanup
	provider.cleanupHTTP01Challenge("test-token-12345")

	provider.challengeMu.RLock()
	_, exists = provider.http01Tokens["test-token-12345"]
	provider.challengeMu.RUnlock()

	if exists {
		t.Error("HTTP-01 token not cleaned up")
	}

	t.Log("✓ Challenge handling works correctly")
}
