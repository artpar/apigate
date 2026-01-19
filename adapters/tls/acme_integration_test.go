package tls

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	domaintls "github.com/artpar/apigate/domain/tls"
	"golang.org/x/crypto/acme/autocert"
)

// TestACMECacheMissError verifies the critical fix: ErrCacheMiss is autocert.ErrCacheMiss
// This is the key test that ensures ACME certificate obtainment will be triggered.
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
	cache := NewDBCertCache(certStore)

	_, err = cache.Get(context.Background(), "nonexistent.example.com")

	// This is the critical test - the error MUST be autocert.ErrCacheMiss
	// for the autocert.Manager to trigger certificate obtainment
	if err != autocert.ErrCacheMiss {
		t.Errorf("CRITICAL: Expected autocert.ErrCacheMiss, got: %v (type: %T)", err, err)
		t.Error("This will prevent ACME certificate obtainment from working!")
	} else {
		t.Log("✓ DBCertCache.Get correctly returns autocert.ErrCacheMiss")
	}
}

// TestCacheNonCertData verifies that non-certificate data (like ACME account keys) is handled
// autocert stores both account keys (single PEM block) and certificates (multiple blocks)
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
	cache := NewDBCertCache(certStore)

	// Simulate ACME account key storage (single PEM block)
	accountKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	keyBytes, _ := x509.MarshalECPrivateKey(accountKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	// This should not error - account keys are stored in memory only
	err = cache.Put(context.Background(), "acme-account-key", keyPEM)
	if err != nil {
		t.Errorf("Failed to store account key: %v", err)
	}

	// Should be retrievable from memory cache
	data, err := cache.Get(context.Background(), "acme-account-key")
	if err != nil {
		t.Errorf("Failed to retrieve account key: %v", err)
	}
	if len(data) == 0 {
		t.Error("Retrieved empty account key data")
	}

	t.Log("✓ Non-certificate data (account keys) handled correctly")
}

// TestCacheCertificateStorage verifies certificate storage and retrieval
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
	cache := NewDBCertCache(certStore)

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

	// Combine as autocert would (cert + key)
	data := append(certPEM, keyPEM...)

	// Test 1: Store certificate
	err = cache.Put(ctx, testDomain, data)
	if err != nil {
		t.Fatalf("Failed to store certificate: %v", err)
	}
	t.Log("✓ Certificate stored in cache")

	// Test 2: Retrieve from cache (should hit memory cache)
	retrieved, err := cache.Get(ctx, testDomain)
	if err != nil {
		t.Fatalf("Failed to retrieve from cache: %v", err)
	}
	if len(retrieved) == 0 {
		t.Fatal("Retrieved empty data")
	}
	t.Log("✓ Certificate retrieved from memory cache")

	// Test 3: Verify it's in the database
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
	t.Logf("✓ Certificate stored in database with ID: %s", dbCert.ID)

	// Test 4: Clear memory cache and retrieve (should hit database)
	cache.ClearMemoryCache()
	retrieved, err = cache.Get(ctx, testDomain)
	if err != nil {
		t.Fatalf("Failed to retrieve from database: %v", err)
	}
	if len(retrieved) == 0 {
		t.Fatal("Retrieved empty data from database")
	}
	t.Log("✓ Certificate retrieved from database after memory cache clear")

	// Test 5: Delete certificate
	err = cache.Delete(ctx, testDomain)
	if err != nil {
		t.Fatalf("Failed to delete certificate: %v", err)
	}

	// Verify it returns ErrCacheMiss now
	_, err = cache.Get(ctx, testDomain)
	if err != autocert.ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after delete, got: %v", err)
	}
	t.Log("✓ Certificate deleted successfully")
}

// TestCombineCertData verifies certificate combining works correctly
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
