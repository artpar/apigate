package tls

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	cryptotls "crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	domaintls "github.com/artpar/apigate/domain/tls"
	"github.com/artpar/apigate/ports"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// LetsEncrypt production directory
	letsEncryptProduction = "https://acme-v02.api.letsencrypt.org/directory"
	// LetsEncrypt staging directory (for testing)
	letsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

// ACMEProvider implements TLS certificate provisioning via ACME (Let's Encrypt).
type ACMEProvider struct {
	certStore   ports.CertificateStore
	cache       *DBCertCache
	manager     *autocert.Manager
	email       string
	staging     bool
	renewalDays int
	logger      *slog.Logger

	mu           sync.RWMutex
	domains      []string
	acmeClient   *acme.Client
	accountKey   crypto.Signer
}

// ACMEConfig holds configuration for ACME provider.
type ACMEConfig struct {
	Email       string
	Staging     bool     // Use staging server for testing
	Domains     []string // Domains to obtain certificates for
	RenewalDays int      // Days before expiry to renew (default: 30)
}

// NewACMEProvider creates a new ACME TLS provider.
// The certStore should implement both CertificateStore and ACMECacheStore interfaces
// for full persistence support. If cacheStore is nil, ACME account keys will only
// be stored in memory (which can cause Let's Encrypt rate limiting on restarts).
func NewACMEProvider(certStore ports.CertificateStore, cfg ACMEConfig) (*ACMEProvider, error) {
	// Try to use certStore as ACMECacheStore if it implements the interface
	var cacheStore ports.ACMECacheStore
	if cs, ok := certStore.(ports.ACMECacheStore); ok {
		cacheStore = cs
	}
	cache := NewDBCertCache(certStore, cacheStore)

	renewalDays := cfg.RenewalDays
	if renewalDays <= 0 {
		renewalDays = 30
	}

	directoryURL := letsEncryptProduction
	if cfg.Staging {
		directoryURL = letsEncryptStaging
	}

	// Generate account key for ACME
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate account key: %w", err)
	}

	// Create HTTP client with appropriate timeouts for ACME operations
	// Without this, HTTP calls to Let's Encrypt could hang indefinitely
	// if there are network issues (this was the root cause of production ACME hangs)
	//
	// Force IPv4 using "tcp4" network to avoid IPv6 connectivity issues.
	// Some servers have IPv6 enabled but broken routing, causing connection
	// attempts to hang when dual-stack (IPv4+IPv6) is used.
	dialer := &net.Dialer{
		Timeout:   10 * time.Second, // Connection timeout
		KeepAlive: 30 * time.Second,
	}
	acmeHTTPClient := &http.Client{
		Timeout: 60 * time.Second, // Overall request timeout
		Transport: &http.Transport{
			// Force IPv4 by using custom dial function with "tcp4" network
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Override network to force IPv4 only
				return dialer.DialContext(ctx, "tcp4", addr)
			},
			ForceAttemptHTTP2:     true,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	provider := &ACMEProvider{
		certStore:   certStore,
		cache:       cache,
		email:       cfg.Email,
		staging:     cfg.Staging,
		renewalDays: renewalDays,
		domains:     cfg.Domains,
		accountKey:  accountKey,
		logger:      slog.Default(),
		acmeClient: &acme.Client{
			DirectoryURL: directoryURL,
			Key:          accountKey,
			HTTPClient:   acmeHTTPClient,
		},
	}

	// Create autocert manager with explicit Client for the correct directory
	// IMPORTANT: Always set Client explicitly - when nil, autocert uses lazy initialization
	// which can fail silently for production ACME. Setting it explicitly ensures
	// the correct directory URL is used from the start.
	// Also set HTTPClient with timeouts to prevent indefinite hangs on network issues.
	// NOTE: We set Client.Key to the same accountKey so autocert doesn't need to
	// generate/load its own key - this avoids potential initialization issues.
	provider.manager = &autocert.Manager{
		Cache:      cache,
		Prompt:     autocert.AcceptTOS,
		Email:      cfg.Email,
		HostPolicy: provider.hostPolicy,
		Client: &acme.Client{
			DirectoryURL: directoryURL,
			HTTPClient:   acmeHTTPClient,
			Key:          accountKey,
		},
	}

	return provider, nil
}

// Name returns the provider name.
func (p *ACMEProvider) Name() string {
	return "acme"
}

// getDirectoryURL returns the ACME directory URL for logging.
func (p *ACMEProvider) getDirectoryURL() string {
	if p.staging {
		return letsEncryptStaging
	}
	return letsEncryptProduction
}

// SetLogger sets a custom logger for the ACME provider.
func (p *ACMEProvider) SetLogger(logger *slog.Logger) {
	p.logger = logger
	p.cache.SetLogger(logger)
}

// GetCertificateWithLogging wraps autocert.Manager.GetCertificate with logging.
// This is the function that should be used in tls.Config.GetCertificate to provide
// visibility into certificate acquisition and failures.
func (p *ACMEProvider) GetCertificateWithLogging(hello *cryptotls.ClientHelloInfo) (*cryptotls.Certificate, error) {
	domain := hello.ServerName
	start := time.Now()

	p.logger.Info("TLS certificate requested",
		"domain", domain,
		"staging", p.staging)

	// Check host policy first for early logging
	if err := p.hostPolicy(context.Background(), domain); err != nil {
		p.logger.Error("domain rejected by host policy",
			"domain", domain,
			"error", err,
			"allowed_domains", p.domains)
		return nil, err
	}

	// Log that we're about to call autocert - this helps identify hangs
	p.logger.Info("calling autocert manager",
		"domain", domain,
		"staging", p.staging,
		"acme_directory", p.getDirectoryURL())

	// Use a channel to implement timeout around GetCertificate
	// autocert.Manager.GetCertificate doesn't accept a context, so we wrap it
	// with a goroutine and timeout to prevent indefinite hangs
	type result struct {
		cert *cryptotls.Certificate
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		p.logger.Info("entering autocert.GetCertificate goroutine",
			"domain", domain)
		cert, err := p.manager.GetCertificate(hello)
		p.logger.Info("autocert.GetCertificate returned",
			"domain", domain,
			"has_cert", cert != nil,
			"has_error", err != nil)
		resultCh <- result{cert, err}
	}()

	// Wait for result with 90 second timeout (longer than HTTP client timeout
	// to allow for full ACME flow including challenge verification)
	select {
	case r := <-resultCh:
		cert, err := r.cert, r.err

		if err != nil {
			p.logger.Error("failed to get certificate",
				"domain", domain,
				"duration", time.Since(start),
				"staging", p.staging,
				"error", err)
			return nil, err
		}

		// Log success with certificate details if available
		if cert != nil && len(cert.Certificate) > 0 {
			if x509Cert, parseErr := x509.ParseCertificate(cert.Certificate[0]); parseErr == nil {
				p.logger.Info("certificate obtained successfully",
					"domain", domain,
					"duration", time.Since(start),
					"issuer", x509Cert.Issuer.CommonName,
					"expires", x509Cert.NotAfter)
			} else {
				p.logger.Info("certificate obtained",
					"domain", domain,
					"duration", time.Since(start))
			}
		} else {
			p.logger.Info("certificate obtained",
				"domain", domain,
				"duration", time.Since(start))
		}

		return cert, nil

	case <-time.After(90 * time.Second):
		p.logger.Error("certificate acquisition timed out",
			"domain", domain,
			"duration", time.Since(start),
			"staging", p.staging,
			"timeout", "90s")
		return nil, fmt.Errorf("certificate acquisition timed out after 90s for domain %s", domain)
	}
}

// hostPolicy checks if a domain is allowed.
func (p *ACMEProvider) hostPolicy(ctx context.Context, host string) error {
	p.mu.RLock()
	domains := p.domains
	p.mu.RUnlock()

	// If no domains configured, allow all
	if len(domains) == 0 {
		return nil
	}

	// Check if host matches any configured domain
	for _, d := range domains {
		if d == host {
			return nil
		}
		// Support wildcard matching
		if len(d) > 1 && d[0] == '*' && d[1] == '.' {
			// *.example.com should match sub.example.com
			suffix := d[1:] // .example.com
			if len(host) > len(suffix) && host[len(host)-len(suffix):] == suffix {
				return nil
			}
		}
	}

	return fmt.Errorf("host %q not in allowed domains", host)
}

// UpdateDomains updates the list of allowed domains.
func (p *ACMEProvider) UpdateDomains(domains []string) {
	p.mu.Lock()
	p.domains = domains
	p.mu.Unlock()
}

// GetCertificate retrieves or obtains a certificate for a domain.
// This is the main method called by tls.Config.GetCertificate.
func (p *ACMEProvider) GetCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	// Try to get from database first
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err == nil && cert.IsActive() && !cert.NeedsRenewal(p.renewalDays) {
		return cert, nil
	}

	// Need to obtain/renew certificate
	return p.ObtainCertificate(ctx, domain)
}

// ObtainCertificate obtains a new certificate for a domain via ACME.
func (p *ACMEProvider) ObtainCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	// Check host policy first
	if err := p.hostPolicy(ctx, domain); err != nil {
		return domaintls.Certificate{}, err
	}

	// Use autocert manager to get the certificate
	// This will either return cached cert or obtain a new one
	hello := &cryptotls.ClientHelloInfo{
		ServerName: domain,
	}

	tlsCert, err := p.manager.GetCertificate(hello)
	if err != nil {
		return domaintls.Certificate{}, fmt.Errorf("get certificate from ACME: %w", err)
	}

	// Parse the certificate to extract metadata
	if len(tlsCert.Certificate) == 0 {
		return domaintls.Certificate{}, errors.New("no certificate returned from ACME")
	}

	_, err = x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return domaintls.Certificate{}, fmt.Errorf("parse certificate: %w", err)
	}

	// Get from database (should have been stored by cache)
	return p.certStore.GetByDomain(ctx, domain)
}

// RenewCertificate forces renewal of a certificate.
func (p *ACMEProvider) RenewCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	// Delete from cache to force re-fetch
	if err := p.cache.Delete(ctx, domain); err != nil {
		// Log but continue - we want to renew anyway
	}

	// Clear memory cache
	p.cache.ClearMemoryCache()

	// Obtain new certificate
	return p.ObtainCertificate(ctx, domain)
}

// RevokeCertificate revokes a certificate.
func (p *ACMEProvider) RevokeCertificate(ctx context.Context, domain string, reason string) error {
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err != nil {
		return fmt.Errorf("get certificate: %w", err)
	}

	// Parse the certificate
	block, _ := pem.Decode(cert.CertPEM)
	if block == nil {
		return errors.New("failed to decode certificate PEM")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	// Revoke via ACME
	err = p.acmeClient.RevokeCert(ctx, nil, x509Cert.Raw, acme.CRLReasonUnspecified)
	if err != nil {
		return fmt.Errorf("revoke certificate via ACME: %w", err)
	}

	// Update status in database
	now := time.Now().UTC()
	cert.Status = domaintls.StatusRevoked
	cert.RevokedAt = &now
	cert.RevokeReason = reason

	if err := p.certStore.Update(ctx, cert); err != nil {
		return fmt.Errorf("update certificate status: %w", err)
	}

	// Remove from cache
	p.cache.Delete(ctx, domain)

	return nil
}

// CheckRenewal checks if a certificate needs renewal.
func (p *ACMEProvider) CheckRenewal(ctx context.Context, domain string, renewalDays int) (bool, error) {
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return true, nil // No certificate, needs obtaining
		}
		return false, fmt.Errorf("get certificate: %w", err)
	}

	return cert.NeedsRenewal(renewalDays), nil
}

// GetManager returns the autocert manager for use in HTTP handlers.
// This can be used to handle ACME HTTP-01 challenges.
func (p *ACMEProvider) GetManager() *autocert.Manager {
	return p.manager
}

// Ensure interface compliance.
var _ ports.TLSProvider = (*ACMEProvider)(nil)

// ClientHelloInfo is a type alias for compatibility.
type ClientHelloInfo = cryptotls.ClientHelloInfo
