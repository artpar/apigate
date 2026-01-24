// Package tls provides TLS certificate management adapters.
package tls

import (
	"context"
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
	return &DBCertCache{
		store:      store,
		cacheStore: cacheStore,
		logger:     slog.Default(),
		cache:      make(map[string][]byte),
		ttl:        5 * time.Minute, // Cache certificates for 5 minutes
	}
}

// SetLogger sets a custom logger for the certificate cache.
func (c *DBCertCache) SetLogger(logger *slog.Logger) {
	c.logger = logger
}

// Get retrieves a certificate data from cache.
// Implements autocert.Cache interface.
func (c *DBCertCache) Get(ctx context.Context, key string) ([]byte, error) {
	isAccountKey := strings.Contains(key, "acme_account") || strings.HasPrefix(key, "+")

	// Check in-memory cache first
	c.mu.RLock()
	if data, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		if isAccountKey {
			c.logger.Debug("ACME account key retrieved from memory cache", "key_prefix", key[:min(20, len(key))])
		} else {
			c.logger.Debug("certificate retrieved from memory cache", "domain", key)
		}
		return data, nil
	}
	c.mu.RUnlock()

	// Account keys should be stored in database for persistence across restarts
	if isAccountKey {
		// Try to retrieve from database if cache store is available
		if c.cacheStore != nil {
			data, err := c.cacheStore.GetCache(ctx, key)
			if err == nil && len(data) > 0 {
				// Found in database, cache in memory for future requests
				c.mu.Lock()
				c.cache[key] = data
				c.mu.Unlock()
				c.logger.Debug("ACME account key retrieved from database", "key_prefix", key[:min(20, len(key))])
				return data, nil
			}
		}
		c.logger.Debug("ACME account key not found (will create new)", "key_prefix", key[:min(20, len(key))])
		return nil, autocert.ErrCacheMiss
	}

	// Key format: domain name
	cert, err := c.store.GetByDomain(ctx, key)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			c.logger.Debug("certificate not in database (will obtain via ACME)", "domain", key)
			return nil, autocert.ErrCacheMiss
		}
		c.logger.Error("failed to get certificate from database", "domain", key, "error", err)
		return nil, fmt.Errorf("get certificate from database: %w", err)
	}

	// Check if certificate is still valid
	if !cert.IsActive() {
		c.logger.Debug("certificate found but not active", "domain", key, "status", cert.Status)
		return nil, autocert.ErrCacheMiss
	}

	// Combine cert and key data (autocert expects this format)
	data := combineCertData(cert)

	// Update in-memory cache
	c.mu.Lock()
	c.cache[key] = data
	c.mu.Unlock()

	c.logger.Debug("certificate retrieved from database", "domain", key, "expires", cert.ExpiresAt)
	return data, nil
}

// Put stores certificate data in cache.
// Implements autocert.Cache interface.
// Note: autocert uses this cache for both ACME account keys and TLS certificates.
// Account keys are single PEM blocks (just private key), while certificates have multiple blocks.
func (c *DBCertCache) Put(ctx context.Context, key string, data []byte) error {
	// Try to parse as certificate data (cert + key)
	certPEM, keyPEM, chainPEM, err := splitCertData(data)
	if err != nil {
		// Not a certificate (probably an ACME account key) - store in database for persistence
		// This ensures account keys survive restarts, preventing Let's Encrypt rate limiting
		if c.cacheStore != nil {
			if err := c.cacheStore.PutCache(ctx, key, data); err != nil {
				c.logger.Error("failed to store ACME account key in database", "key_prefix", key[:min(20, len(key))], "error", err)
				// Fall back to memory-only storage
			} else {
				c.logger.Debug("ACME account key stored in database", "key_prefix", key[:min(20, len(key))], "data_len", len(data))
			}
		}
		// Also store in memory cache for fast access
		c.mu.Lock()
		c.cache[key] = data
		c.mu.Unlock()
		return nil
	}

	// Parse certificate to extract metadata
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	now := time.Now().UTC()

	// Check if certificate already exists
	existing, err := c.store.GetByDomain(ctx, key)
	if err == nil && existing.ID != "" {
		// Update existing certificate
		existing.CertPEM = certPEM
		existing.KeyPEM = keyPEM
		existing.ChainPEM = chainPEM
		existing.IssuedAt = parsedCert.NotBefore
		existing.ExpiresAt = parsedCert.NotAfter
		existing.Issuer = parsedCert.Issuer.CommonName
		existing.SerialNumber = parsedCert.SerialNumber.String()
		existing.Status = domaintls.StatusActive
		existing.UpdatedAt = now

		if err := c.store.Update(ctx, existing); err != nil {
			c.logger.Error("failed to update certificate in database", "domain", key, "error", err)
			return fmt.Errorf("update certificate in database: %w", err)
		}
		c.logger.Info("certificate renewed and stored", "domain", key, "issuer", parsedCert.Issuer.CommonName, "expires", parsedCert.NotAfter)
	} else {
		// Create new certificate
		cert := domaintls.Certificate{
			ID:           domaintls.GenerateCertificateID(),
			Domain:       key,
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

		if err := c.store.Create(ctx, cert); err != nil {
			c.logger.Error("failed to store certificate in database", "domain", key, "error", err)
			return fmt.Errorf("store certificate in database: %w", err)
		}
		c.logger.Info("new certificate obtained and stored", "domain", key, "issuer", parsedCert.Issuer.CommonName, "expires", parsedCert.NotAfter)
	}

	// Update in-memory cache
	c.mu.Lock()
	c.cache[key] = data
	c.mu.Unlock()

	return nil
}

// Delete removes certificate data from cache.
// Implements autocert.Cache interface.
func (c *DBCertCache) Delete(ctx context.Context, key string) error {
	// Remove from in-memory cache
	c.mu.Lock()
	delete(c.cache, key)
	c.mu.Unlock()

	// Get the certificate by domain
	cert, err := c.store.GetByDomain(ctx, key)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil // Already doesn't exist
		}
		return fmt.Errorf("get certificate: %w", err)
	}

	// Delete from database
	if err := c.store.Delete(ctx, cert.ID); err != nil {
		return fmt.Errorf("delete certificate from database: %w", err)
	}

	return nil
}

// ClearMemoryCache clears the in-memory cache.
// Useful when certificates are updated externally.
func (c *DBCertCache) ClearMemoryCache() {
	c.mu.Lock()
	c.cache = make(map[string][]byte)
	c.mu.Unlock()
}

// combineCertData combines certificate, key, and chain into a single blob.
// Format: cert PEM + key PEM + chain PEM (if present)
func combineCertData(cert domaintls.Certificate) []byte {
	data := make([]byte, 0, len(cert.CertPEM)+len(cert.KeyPEM)+len(cert.ChainPEM)+2)
	data = append(data, cert.CertPEM...)
	data = append(data, '\n')
	data = append(data, cert.KeyPEM...)
	if len(cert.ChainPEM) > 0 {
		data = append(data, '\n')
		data = append(data, cert.ChainPEM...)
	}
	return data
}

// splitCertData splits combined certificate data back into components.
func splitCertData(data []byte) (certPEM, keyPEM, chainPEM []byte, err error) {
	// Parse all PEM blocks
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

	// First block should be the certificate
	// Look for the private key block
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

	// Encode the main certificate
	certPEM = pem.EncodeToMemory(certBlocks[0])

	// Encode the private key
	keyPEM = pem.EncodeToMemory(keyBlock)

	// Encode any additional certificates as chain
	if len(certBlocks) > 1 {
		for _, block := range certBlocks[1:] {
			chainPEM = append(chainPEM, pem.EncodeToMemory(block)...)
		}
	}

	return certPEM, keyPEM, chainPEM, nil
}
