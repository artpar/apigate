// Package tls provides TLS certificate management adapters.
package tls

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	domaintls "github.com/artpar/apigate/domain/tls"
	"github.com/artpar/apigate/ports"
	"golang.org/x/crypto/acme/autocert"
)

// DBCertCache implements autocert.Cache using database storage.
// This enables horizontal scaling - all instances share the same certificates.
type DBCertCache struct {
	store      ports.CertificateStore
	cacheStore ports.ACMECacheStore // For ACME account keys and other cache data
	logger     *slog.Logger

	// In-memory cache for performance
	mu    sync.RWMutex
	cache map[string][]byte
	ttl   time.Duration
}

// NewDBCertCache creates a new database-backed certificate cache.
// The cacheStore is used for ACME account keys and other autocert cache data.
// If cacheStore is nil, account keys will only be stored in memory (not recommended).
func NewDBCertCache(store ports.CertificateStore, cacheStore ports.ACMECacheStore) *DBCertCache {
	logger := slog.Default()
	logger.Info("[CACHE:INIT] Creating DBCertCache",
		"has_cert_store", store != nil,
		"has_acme_cache_store", cacheStore != nil,
		"ttl", "5m")

	return &DBCertCache{
		store:      store,
		cacheStore: cacheStore,
		logger:     logger,
		cache:      make(map[string][]byte),
		ttl:        5 * time.Minute,
	}
}

// SetLogger sets a custom logger for the certificate cache.
func (c *DBCertCache) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// Get retrieves a certificate data from cache.
// Implements autocert.Cache interface.
func (c *DBCertCache) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	isAccountKey := strings.Contains(key, "acme_account") || strings.HasPrefix(key, "+")

	c.logger.Info("[CACHE:GET] Cache.Get called",
		"key", truncateKey(key),
		"is_account_key", isAccountKey)

	// Check in-memory cache first
	c.mu.RLock()
	data, inMemory := c.cache[key]
	c.mu.RUnlock()

	if inMemory {
		c.logger.Info("[CACHE:GET] Found in memory cache",
			"key", truncateKey(key),
			"data_len", len(data),
			"duration", time.Since(start))
		return data, nil
	}

	c.logger.Debug("[CACHE:GET] Not in memory cache, checking database",
		"key", truncateKey(key))

	// Account keys should be stored in database for persistence across restarts
	if isAccountKey {
		c.logger.Info("[CACHE:GET] Looking up ACME account key in database",
			"key", truncateKey(key),
			"has_cache_store", c.cacheStore != nil)

		if c.cacheStore != nil {
			dbStart := time.Now()
			data, err := c.cacheStore.GetCache(ctx, key)
			dbDuration := time.Since(dbStart)

			if err == nil && len(data) > 0 {
				c.mu.Lock()
				c.cache[key] = data
				c.mu.Unlock()

				c.logger.Info("[CACHE:GET] ACME account key retrieved from database",
					"key", truncateKey(key),
					"data_len", len(data),
					"db_duration", dbDuration,
					"total_duration", time.Since(start))
				return data, nil
			}

			if err != nil {
				c.logger.Warn("[CACHE:GET] Failed to retrieve ACME account key from database",
					"key", truncateKey(key),
					"error", err,
					"db_duration", dbDuration)
			} else {
				c.logger.Info("[CACHE:GET] ACME account key not found in database (empty result)",
					"key", truncateKey(key),
					"db_duration", dbDuration)
			}
		} else {
			c.logger.Warn("[CACHE:GET] No cache store available for ACME account keys")
		}

		c.logger.Info("[CACHE:GET] ACME account key not found - autocert will create new",
			"key", truncateKey(key),
			"duration", time.Since(start))
		return nil, autocert.ErrCacheMiss
	}

	// Extract domain from autocert key format (domain+rsa, domain+ecdsa, or just domain)
	domain := key
	if idx := strings.LastIndex(key, "+"); idx > 0 {
		suffix := key[idx:]
		if suffix == "+rsa" || suffix == "+ecdsa" {
			domain = key[:idx]
		}
	}

	c.logger.Info("[CACHE:GET] Looking up certificate in database",
		"key", key,
		"domain", domain)

	dbStart := time.Now()
	cert, err := c.store.GetByDomain(ctx, domain)
	dbDuration := time.Since(dbStart)

	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			c.logger.Info("[CACHE:GET] certificate not in database - will obtain via ACME",
				"domain", domain,
				"db_duration", dbDuration,
				"total_duration", time.Since(start))
			return nil, autocert.ErrCacheMiss
		}
		c.logger.Error("[CACHE:GET] Database error getting certificate",
			"domain", domain,
			"error", err,
			"db_duration", dbDuration)
		return nil, fmt.Errorf("get certificate from database: %w", err)
	}

	c.logger.Info("[CACHE:GET] Certificate found in database",
		"domain", domain,
		"status", cert.Status,
		"expires", cert.ExpiresAt,
		"issuer", cert.Issuer,
		"db_duration", dbDuration)

	// Check if certificate is still valid
	if !cert.IsActive() {
		c.logger.Warn("[CACHE:GET] Certificate found but not active",
			"domain", domain,
			"status", cert.Status,
			"duration", time.Since(start))
		return nil, autocert.ErrCacheMiss
	}

	// Combine cert and key data
	data = combineCertData(cert)

	// Check key type matching
	wantsRSA := strings.HasSuffix(key, "+rsa")
	wantsECDSA := key == domain || strings.HasSuffix(key, "+ecdsa")

	c.logger.Debug("[CACHE:GET] Checking key type compatibility",
		"key", key,
		"wants_rsa", wantsRSA,
		"wants_ecdsa", wantsECDSA)

	if wantsECDSA || wantsRSA {
		block, _ := pem.Decode(cert.KeyPEM)
		if block != nil {
			_, rsaErr := x509.ParsePKCS1PrivateKey(block.Bytes)
			_, pkcs8Key, _ := func() (interface{}, interface{}, error) {
				k, e := x509.ParsePKCS8PrivateKey(block.Bytes)
				return k, k, e
			}()

			isRSA := rsaErr == nil
			if pkcs8Key != nil {
				_, isRSA = pkcs8Key.(*rsa.PrivateKey)
			}

			c.logger.Debug("[CACHE:GET] Key type analysis",
				"domain", domain,
				"key_is_rsa", isRSA,
				"wants_rsa", wantsRSA,
				"wants_ecdsa", wantsECDSA)

			if wantsECDSA && isRSA {
				c.logger.Info("[CACHE:GET] Key type mismatch - want ECDSA, have RSA",
					"domain", domain,
					"duration", time.Since(start))
				return nil, autocert.ErrCacheMiss
			}
			if wantsRSA && !isRSA {
				c.logger.Info("[CACHE:GET] Key type mismatch - want RSA, have ECDSA",
					"domain", domain,
					"duration", time.Since(start))
				return nil, autocert.ErrCacheMiss
			}
		}
	}

	// Update in-memory cache
	c.mu.Lock()
	c.cache[key] = data
	c.mu.Unlock()

	c.logger.Info("[CACHE:GET] Certificate retrieved successfully",
		"domain", domain,
		"data_len", len(data),
		"expires", cert.ExpiresAt,
		"total_duration", time.Since(start))

	return data, nil
}

// Put stores certificate data in cache.
// Implements autocert.Cache interface.
func (c *DBCertCache) Put(ctx context.Context, key string, data []byte) error {
	start := time.Now()

	c.logger.Info("[CACHE:PUT] Cache.Put called",
		"key", truncateKey(key),
		"data_len", len(data))

	// Try to parse as certificate data (cert + key)
	certPEM, keyPEM, chainPEM, err := splitCertData(data)
	if err != nil {
		// Not a certificate (probably an ACME account key)
		c.logger.Info("[CACHE:PUT] Data is not a certificate (likely ACME account key)",
			"key", truncateKey(key),
			"parse_error", err)

		if c.cacheStore != nil {
			dbStart := time.Now()
			if err := c.cacheStore.PutCache(ctx, key, data); err != nil {
				c.logger.Error("[CACHE:PUT] Failed to store ACME account key in database",
					"key", truncateKey(key),
					"error", err,
					"db_duration", time.Since(dbStart))
			} else {
				c.logger.Info("[CACHE:PUT] ACME account key stored in database",
					"key", truncateKey(key),
					"data_len", len(data),
					"db_duration", time.Since(dbStart))
			}
		} else {
			c.logger.Warn("[CACHE:PUT] No cache store - account key stored in memory only",
				"key", truncateKey(key))
		}

		c.mu.Lock()
		c.cache[key] = data
		c.mu.Unlock()

		c.logger.Info("[CACHE:PUT] Account key cached in memory",
			"key", truncateKey(key),
			"duration", time.Since(start))
		return nil
	}

	c.logger.Info("[CACHE:PUT] Parsed certificate data",
		"key", key,
		"cert_len", len(certPEM),
		"key_len", len(keyPEM),
		"chain_len", len(chainPEM))

	// Parse certificate to extract metadata
	block, _ := pem.Decode(certPEM)
	if block == nil {
		c.logger.Error("[CACHE:PUT] Failed to decode certificate PEM",
			"key", key)
		return fmt.Errorf("failed to decode certificate PEM")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		c.logger.Error("[CACHE:PUT] Failed to parse certificate",
			"key", key,
			"error", err)
		return fmt.Errorf("parse certificate: %w", err)
	}

	c.logger.Info("[CACHE:PUT] Certificate parsed",
		"key", key,
		"subject", parsedCert.Subject.CommonName,
		"issuer", parsedCert.Issuer.CommonName,
		"issuer_org", parsedCert.Issuer.Organization,
		"not_before", parsedCert.NotBefore,
		"not_after", parsedCert.NotAfter,
		"serial", parsedCert.SerialNumber.String(),
		"dns_names", parsedCert.DNSNames)

	now := time.Now().UTC()

	// Extract domain from autocert key format
	domain := key
	if idx := strings.LastIndex(key, "+"); idx > 0 {
		suffix := key[idx:]
		if suffix == "+rsa" || suffix == "+ecdsa" {
			domain = key[:idx]
		}
	}

	// Check if certificate already exists
	existing, err := c.store.GetByDomain(ctx, domain)
	if err == nil && existing.ID != "" {
		c.logger.Info("[CACHE:PUT] Updating existing certificate",
			"domain", domain,
			"existing_id", existing.ID,
			"old_expires", existing.ExpiresAt,
			"new_expires", parsedCert.NotAfter)

		existing.CertPEM = certPEM
		existing.KeyPEM = keyPEM
		existing.ChainPEM = chainPEM
		existing.IssuedAt = parsedCert.NotBefore
		existing.ExpiresAt = parsedCert.NotAfter
		existing.Issuer = parsedCert.Issuer.CommonName
		existing.SerialNumber = parsedCert.SerialNumber.String()
		existing.Status = domaintls.StatusActive
		existing.UpdatedAt = now

		dbStart := time.Now()
		if err := c.store.Update(ctx, existing); err != nil {
			c.logger.Error("[CACHE:PUT] Failed to update certificate in database",
				"domain", domain,
				"error", err,
				"db_duration", time.Since(dbStart))
			return fmt.Errorf("update certificate in database: %w", err)
		}

		c.logger.Info("[CACHE:PUT] Certificate renewed and stored",
			"domain", domain,
			"issuer", parsedCert.Issuer.CommonName,
			"expires", parsedCert.NotAfter,
			"db_duration", time.Since(dbStart),
			"total_duration", time.Since(start))
	} else {
		c.logger.Info("[CACHE:PUT] Creating new certificate record",
			"domain", domain,
			"lookup_error", err)

		cert := domaintls.Certificate{
			ID:           domaintls.GenerateCertificateID(),
			Domain:       domain,
			CertPEM:      certPEM,
			ChainPEM:     chainPEM,
			KeyPEM:       keyPEM,
			IssuedAt:     parsedCert.NotBefore,
			ExpiresAt:    parsedCert.NotAfter,
			Issuer:       parsedCert.Issuer.CommonName,
			SerialNumber: parsedCert.SerialNumber.String(),
			Status:       domaintls.StatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		dbStart := time.Now()
		if err := c.store.Create(ctx, cert); err != nil {
			c.logger.Error("[CACHE:PUT] Failed to store certificate in database",
				"domain", domain,
				"error", err,
				"db_duration", time.Since(dbStart))
			return fmt.Errorf("store certificate in database: %w", err)
		}

		c.logger.Info("[CACHE:PUT] New certificate obtained and stored",
			"domain", domain,
			"cert_id", cert.ID,
			"issuer", parsedCert.Issuer.CommonName,
			"expires", parsedCert.NotAfter,
			"db_duration", time.Since(dbStart),
			"total_duration", time.Since(start))
	}

	// Update in-memory cache
	c.mu.Lock()
	c.cache[key] = data
	c.mu.Unlock()

	c.logger.Info("[CACHE:PUT] Certificate cached in memory",
		"domain", domain,
		"total_duration", time.Since(start))

	return nil
}

// Delete removes certificate data from cache.
// Implements autocert.Cache interface.
func (c *DBCertCache) Delete(ctx context.Context, key string) error {
	start := time.Now()

	c.logger.Info("[CACHE:DELETE] Cache.Delete called",
		"key", truncateKey(key))

	// Remove from in-memory cache
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()

	c.logger.Debug("[CACHE:DELETE] Removed from memory cache",
		"key", truncateKey(key))

	// Extract domain from autocert key format
	domain := key
	if idx := strings.LastIndex(key, "+"); idx > 0 {
		suffix := key[idx:]
		if suffix == "+rsa" || suffix == "+ecdsa" {
			domain = key[:idx]
		}
	}

	// Get the certificate by domain
	cert, err := c.store.GetByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			c.logger.Info("[CACHE:DELETE] Certificate not found in database (already deleted?)",
				"domain", domain,
				"duration", time.Since(start))
			return nil
		}
		c.logger.Error("[CACHE:DELETE] Error looking up certificate",
			"domain", domain,
			"error", err)
		return fmt.Errorf("get certificate: %w", err)
	}

	// Delete from database
	dbStart := time.Now()
	if err := c.store.Delete(ctx, cert.ID); err != nil {
		c.logger.Error("[CACHE:DELETE] Failed to delete certificate from database",
			"domain", domain,
			"cert_id", cert.ID,
			"error", err,
			"db_duration", time.Since(dbStart))
		return fmt.Errorf("delete certificate from database: %w", err)
	}

	c.logger.Info("[CACHE:DELETE] Certificate deleted from database",
		"domain", domain,
		"cert_id", cert.ID,
		"db_duration", time.Since(dbStart),
		"total_duration", time.Since(start))

	return nil
}

// ClearMemoryCache clears the in-memory cache.
func (c *DBCertCache) ClearMemoryCache() {
	c.mu.Lock()
	count := len(c.cache)
	c.cache = make(map[string][]byte)
	c.mu.Unlock()

	c.logger.Info("[CACHE:CLEAR] Memory cache cleared",
		"entries_cleared", count)
}

// truncateKey returns a truncated key for logging (to avoid logging full keys)
func truncateKey(key string) string {
	if len(key) <= 60 {
		return key
	}
	return key[:60] + "..."
}

// combineCertData combines certificate, key, and chain into a single blob.
func combineCertData(cert domaintls.Certificate) []byte {
	data := make([]byte, 0, len(cert.CertPEM)+len(cert.KeyPEM)+len(cert.ChainPEM)+2)
	data = append(data, cert.KeyPEM...)
	data = append(data, '\n')
	data = append(data, cert.CertPEM...)
	if len(cert.ChainPEM) > 0 {
		data = append(data, '\n')
		data = append(data, cert.ChainPEM...)
	}
	return data
}

// splitCertData splits combined certificate data back into components.
func splitCertData(data []byte) (certPEM, keyPEM, chainPEM []byte, err error) {
	var blocks []*pem.Block
	remaining := data
	for {
		var block *pem.Block
		block, remaining = pem.Decode(remaining)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
	}

	if len(blocks) < 2 {
		return nil, nil, nil, fmt.Errorf("expected at least 2 PEM blocks, got %d", len(blocks))
	}

	var certBlocks []*pem.Block
	var keyBlock *pem.Block

	for _, block := range blocks {
		switch block.Type {
		case "RSA PRIVATE KEY", "EC PRIVATE KEY", "PRIVATE KEY":
			keyBlock = block
		case "CERTIFICATE":
			certBlocks = append(certBlocks, block)
		}
	}

	if keyBlock == nil {
		return nil, nil, nil, fmt.Errorf("no private key found")
	}
	if len(certBlocks) == 0 {
		return nil, nil, nil, fmt.Errorf("no certificate found")
	}

	certPEM = pem.EncodeToMemory(certBlocks[0])
	keyPEM = pem.EncodeToMemory(keyBlock)

	if len(certBlocks) > 1 {
		for _, block := range certBlocks[1:] {
			chainPEM = append(chainPEM, pem.EncodeToMemory(block)...)
		}
	}

	return certPEM, keyPEM, chainPEM, nil
}
