// Package e2e provides end-to-end tests for the complete APIGate flow.
package e2e

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/artpar/apigate/adapters/sqlite"
	adapterstls "github.com/artpar/apigate/adapters/tls"
	"github.com/artpar/apigate/bootstrap"
	"golang.org/x/crypto/acme/autocert"
)

// TestE2E_ACMEPersistence_AccountKey tests that ACME account keys persist across app restarts.
// This is critical to prevent Let's Encrypt rate limiting (Issue #47).
func TestE2E_ACMEPersistence_AccountKey(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Phase 1: Start app, store account key, shutdown
	t.Run("Phase1_StoreAccountKey", func(t *testing.T) {
		db, cleanup := setupACMETestDB(t, dbPath)
		defer cleanup()

		certStore := sqlite.NewCertificateStore(db)
		cache := adapterstls.NewDBCertCache(certStore, certStore)
		ctx := context.Background()

		// Generate ACME account key (simulates what autocert does)
		accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		keyBytes, _ := x509.MarshalECPrivateKey(accountKey)
		keyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		})

		// Store account key using the same key format autocert uses
		accountKeyName := "+acme_account+https://acme-v02.api.letsencrypt.org/directory"
		if err := cache.Put(ctx, accountKeyName, keyPEM); err != nil {
			t.Fatalf("store account key: %v", err)
		}

		t.Log("✓ Account key stored")
	})

	// Phase 2: Start NEW app instance (simulates restart), verify account key persists
	t.Run("Phase2_VerifyAccountKeyPersists", func(t *testing.T) {
		// Open same database (simulates restart)
		db, err := sqlite.Open(dbPath)
		if err != nil {
			t.Fatalf("reopen db: %v", err)
		}
		defer db.Close()

		certStore := sqlite.NewCertificateStore(db)
		cache := adapterstls.NewDBCertCache(certStore, certStore)
		ctx := context.Background()

		accountKeyName := "+acme_account+https://acme-v02.api.letsencrypt.org/directory"

		// Memory cache is empty (new instance), should retrieve from database
		data, err := cache.Get(ctx, accountKeyName)
		if err != nil {
			t.Fatalf("CRITICAL: Account key not found after restart: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("CRITICAL: Account key data is empty after restart")
		}

		// Verify it's valid PEM
		block, _ := pem.Decode(data)
		if block == nil {
			t.Fatal("CRITICAL: Retrieved data is not valid PEM")
		}
		if block.Type != "EC PRIVATE KEY" {
			t.Fatalf("unexpected PEM type: %s", block.Type)
		}

		t.Log("✓ Account key persisted and retrieved after simulated restart")
	})
}

// TestE2E_ACMEPersistence_Certificate tests that TLS certificates persist across app restarts.
func TestE2E_ACMEPersistence_Certificate(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	testDomain := "test.example.com"

	var originalSerial string

	// Phase 1: Store certificate
	t.Run("Phase1_StoreCertificate", func(t *testing.T) {
		db, cleanup := setupACMETestDB(t, dbPath)
		defer cleanup()

		certStore := sqlite.NewCertificateStore(db)
		cache := adapterstls.NewDBCertCache(certStore, certStore)
		ctx := context.Background()

		// Generate self-signed certificate (simulates what ACME would obtain)
		certPEM, keyPEM, serial := generateTestCert(t, testDomain)
		originalSerial = serial
		data := append(certPEM, keyPEM...)

		if err := cache.Put(ctx, testDomain, data); err != nil {
			t.Fatalf("store certificate: %v", err)
		}

		t.Logf("✓ Certificate stored (serial: %s)", serial)
	})

	// Phase 2: Verify certificate persists after restart
	t.Run("Phase2_VerifyCertificatePersists", func(t *testing.T) {
		db, err := sqlite.Open(dbPath)
		if err != nil {
			t.Fatalf("reopen db: %v", err)
		}
		defer db.Close()

		certStore := sqlite.NewCertificateStore(db)
		cache := adapterstls.NewDBCertCache(certStore, certStore)
		ctx := context.Background()

		// Memory cache is empty, should retrieve from database
		data, err := cache.Get(ctx, testDomain)
		if err != nil {
			if err == autocert.ErrCacheMiss {
				t.Fatal("CRITICAL: Certificate not found after restart (ErrCacheMiss)")
			}
			t.Fatalf("CRITICAL: Failed to retrieve certificate: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("CRITICAL: Certificate data is empty after restart")
		}

		// Parse and verify it's the same certificate
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" {
			t.Fatal("retrieved data is not a valid certificate PEM")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("parse certificate: %v", err)
		}

		if cert.SerialNumber.String() != originalSerial {
			t.Errorf("serial mismatch: got %s, want %s", cert.SerialNumber.String(), originalSerial)
		}
		if len(cert.DNSNames) == 0 || cert.DNSNames[0] != testDomain {
			t.Errorf("domain mismatch: got %v, want [%s]", cert.DNSNames, testDomain)
		}

		t.Logf("✓ Certificate persisted and verified (serial: %s, domain: %s)", cert.SerialNumber.String(), testDomain)
	})
}

// TestE2E_ACMEPersistence_FullBootstrap tests certificate persistence with full app bootstrap.
// This tests the actual production code path.
func TestE2E_ACMEPersistence_FullBootstrap(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	// Phase 1: Bootstrap app, store data directly in DB
	t.Run("Phase1_StoreViaBootstrappedApp", func(t *testing.T) {
		// Pre-create database
		db, err := sqlite.Open(dbPath)
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		if err := db.Migrate(); err != nil {
			t.Fatalf("migrate: %v", err)
		}

		// Store account key and certificate directly
		certStore := sqlite.NewCertificateStore(db)
		ctx := context.Background()

		// Store account key
		accountKey := []byte("-----BEGIN EC PRIVATE KEY-----\ntest-key-data\n-----END EC PRIVATE KEY-----")
		if err := certStore.PutCache(ctx, "+acme_account+test", accountKey); err != nil {
			t.Fatalf("store account key: %v", err)
		}

		db.Close()
		t.Log("✓ Data stored in database")
	})

	// Phase 2: Bootstrap full app and verify data accessible
	t.Run("Phase2_VerifyViaBootstrappedApp", func(t *testing.T) {
		os.Setenv(bootstrap.EnvDatabaseDSN, dbPath)
		os.Setenv(bootstrap.EnvLogLevel, "error")
		os.Setenv(bootstrap.EnvLogFormat, "json")
		defer os.Unsetenv(bootstrap.EnvDatabaseDSN)
		defer os.Unsetenv(bootstrap.EnvLogLevel)
		defer os.Unsetenv(bootstrap.EnvLogFormat)

		app, err := bootstrap.New()
		if err != nil {
			t.Fatalf("bootstrap app: %v", err)
		}
		defer app.Shutdown()

		// Verify data by querying database directly
		certStore := sqlite.NewCertificateStore(app.DB)
		ctx := context.Background()

		data, err := certStore.GetCache(ctx, "+acme_account+test")
		if err != nil {
			t.Fatalf("CRITICAL: Account key not accessible via bootstrapped app: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("CRITICAL: Account key data is empty")
		}

		t.Log("✓ Data accessible via fully bootstrapped app")
	})
}

// TestE2E_ACMEPersistence_CacheMissBehavior verifies correct ErrCacheMiss for missing certs.
// This is critical for autocert to know when to obtain new certificates.
func TestE2E_ACMEPersistence_CacheMissBehavior(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"

	db, cleanup := setupACMETestDB(t, dbPath)
	defer cleanup()

	certStore := sqlite.NewCertificateStore(db)
	cache := adapterstls.NewDBCertCache(certStore, certStore)
	ctx := context.Background()

	// Request non-existent certificate
	_, err := cache.Get(ctx, "nonexistent.example.com")
	if err != autocert.ErrCacheMiss {
		t.Fatalf("CRITICAL: Expected autocert.ErrCacheMiss, got: %v (type: %T)", err, err)
	}

	t.Log("✓ Correctly returns autocert.ErrCacheMiss for missing certificates")
}

// TestE2E_ACMEPersistence_MultipleRestarts tests data survives multiple restart cycles.
func TestE2E_ACMEPersistence_MultipleRestarts(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	testDomain := "multi-restart.example.com"

	// Initial setup
	db, cleanup := setupACMETestDB(t, dbPath)
	certStore := sqlite.NewCertificateStore(db)
	cache := adapterstls.NewDBCertCache(certStore, certStore)
	ctx := context.Background()

	certPEM, keyPEM, originalSerial := generateTestCert(t, testDomain)
	data := append(certPEM, keyPEM...)
	if err := cache.Put(ctx, testDomain, data); err != nil {
		t.Fatalf("store certificate: %v", err)
	}
	cleanup()

	// Simulate 3 restart cycles
	for i := 1; i <= 3; i++ {
		t.Run("Restart"+string(rune('0'+i)), func(t *testing.T) {
			db, err := sqlite.Open(dbPath)
			if err != nil {
				t.Fatalf("open db (restart %d): %v", i, err)
			}

			certStore := sqlite.NewCertificateStore(db)
			cache := adapterstls.NewDBCertCache(certStore, certStore)

			data, err := cache.Get(ctx, testDomain)
			if err != nil {
				t.Fatalf("CRITICAL: Certificate lost after restart %d: %v", i, err)
			}

			block, _ := pem.Decode(data)
			cert, _ := x509.ParseCertificate(block.Bytes)
			if cert.SerialNumber.String() != originalSerial {
				t.Fatalf("serial mismatch after restart %d", i)
			}

			db.Close()
			t.Logf("✓ Restart %d: certificate verified", i)
		})
	}
}

// Helper functions

func setupACMETestDB(t *testing.T, dbPath string) (*sqlite.DB, func()) {
	t.Helper()

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("migrate: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func generateTestCert(t *testing.T, domain string) (certPEM, keyPEM []byte, serial string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	serialNumber, _ := rand.Int(rand.Reader, big.NewInt(1<<62))

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		DNSNames:     []string{domain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyBytes, _ := x509.MarshalECPrivateKey(key)
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	return certPEM, keyPEM, serialNumber.String()
}
