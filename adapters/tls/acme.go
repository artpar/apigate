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
	"regexp"
	"strings"
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

// loggingRoundTripper wraps an http.RoundTripper to log ACME requests/responses.
type loggingRoundTripper struct {
	wrapped http.RoundTripper
	logger  *slog.Logger
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	l.logger.Info("[ACME:HTTP:REQ]",
		"method", req.Method,
		"url", req.URL.String(),
		"content_type", req.Header.Get("Content-Type"))

	resp, err := l.wrapped.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		l.logger.Error("[ACME:HTTP:ERR]",
			"method", req.Method,
			"url", req.URL.String(),
			"duration", duration,
			"error", err)
		return nil, err
	}

	l.logger.Info("[ACME:HTTP:RESP]",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"duration", duration,
		"content_type", resp.Header.Get("Content-Type"),
		"retry_after", resp.Header.Get("Retry-After"))

	return resp, nil
}

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

	// Rate limit tracking per domain
	rateLimitMu    sync.RWMutex
	rateLimitUntil map[string]time.Time // domain -> retry after time

	// Challenge tokens for HTTP-01 (used by direct ACME flow)
	challengeMu     sync.RWMutex
	challengeTokens map[string]string // token -> key authorization
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
	logger := slog.Default()

	logger.Info("[ACME:INIT] Starting ACME provider initialization",
		"email", cfg.Email,
		"staging", cfg.Staging,
		"domains", cfg.Domains,
		"renewal_days", cfg.RenewalDays)

	// Try to use certStore as ACMECacheStore if it implements the interface
	var cacheStore ports.ACMECacheStore
	if cs, ok := certStore.(ports.ACMECacheStore); ok {
		cacheStore = cs
		logger.Info("[ACME:INIT] CertStore implements ACMECacheStore interface")
	} else {
		logger.Warn("[ACME:INIT] CertStore does NOT implement ACMECacheStore - account keys will be memory-only")
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
	logger.Info("[ACME:INIT] Using ACME directory",
		"url", directoryURL,
		"is_staging", cfg.Staging)

	// Generate account key for ACME
	logger.Info("[ACME:INIT] Generating ECDSA P-256 account key...")
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		logger.Error("[ACME:INIT] Failed to generate account key", "error", err)
		return nil, fmt.Errorf("generate account key: %w", err)
	}
	logger.Info("[ACME:INIT] Account key generated successfully")

	// Create HTTP client with appropriate timeouts for ACME operations
	logger.Info("[ACME:INIT] Creating HTTP client with timeouts",
		"overall_timeout", "60s",
		"dial_timeout", "10s",
		"tls_handshake_timeout", "10s",
		"response_header_timeout", "30s",
		"force_ipv4", true)

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	// Create a logging round tripper to see ACME request/response
	loggingTransport := &loggingRoundTripper{
		wrapped: &http.Transport{
			// Force IPv4 by using custom dial function with "tcp4" network
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				logger.Debug("[ACME:HTTP] Dialing", "network", "tcp4", "addr", addr)
				conn, err := dialer.DialContext(ctx, "tcp4", addr)
				if err != nil {
					logger.Error("[ACME:HTTP] Dial failed", "addr", addr, "error", err)
				} else {
					logger.Debug("[ACME:HTTP] Dial succeeded", "addr", addr, "local", conn.LocalAddr(), "remote", conn.RemoteAddr())
				}
				return conn, err
			},
			ForceAttemptHTTP2:     true,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          10,
			IdleConnTimeout:       90 * time.Second,
		},
		logger: logger,
	}
	acmeHTTPClient := &http.Client{
		Timeout:   60 * time.Second,
		Transport: loggingTransport,
	}
	logger.Info("[ACME:INIT] HTTP client created")

	provider := &ACMEProvider{
		certStore:       certStore,
		cache:           cache,
		email:           cfg.Email,
		staging:         cfg.Staging,
		renewalDays:     renewalDays,
		domains:         cfg.Domains,
		accountKey:      accountKey,
		logger:          logger,
		rateLimitUntil:  make(map[string]time.Time),
		challengeTokens: make(map[string]string),
		acmeClient: &acme.Client{
			DirectoryURL: directoryURL,
			Key:          accountKey,
			HTTPClient:   acmeHTTPClient,
		},
	}

	// Create the ACME client that will be used by autocert
	logger.Info("[ACME:INIT] Creating ACME client for autocert manager",
		"directory_url", directoryURL)
	acmeClient := &acme.Client{
		DirectoryURL: directoryURL,
		HTTPClient:   acmeHTTPClient,
		Key:          accountKey,
	}

	// Pre-fetch the ACME directory to ensure the client is fully initialized
	logger.Info("[ACME:INIT] Pre-fetching ACME directory (30s timeout)...",
		"url", directoryURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	discoverStart := time.Now()
	dir, err := acmeClient.Discover(ctx)
	discoverDuration := time.Since(discoverStart)

	if err != nil {
		logger.Error("[ACME:INIT] Failed to fetch ACME directory",
			"url", directoryURL,
			"duration", discoverDuration,
			"error", err)
		return nil, fmt.Errorf("failed to fetch ACME directory from %s: %w", directoryURL, err)
	}
	logger.Info("[ACME:INIT] ACME directory fetched successfully",
		"duration", discoverDuration,
		"directory_url", directoryURL,
		"authz_url", dir.AuthzURL,
		"order_url", dir.OrderURL,
		"revoke_url", dir.RevokeURL,
		"nonce_url", dir.NonceURL,
		"terms_url", dir.Terms)

	// Pre-register the ACME account to avoid autocert doing it lazily
	// This is where production might hang - now we'll see it during init
	logger.Info("[ACME:INIT] Pre-registering ACME account (30s timeout)...",
		"email", cfg.Email,
		"directory", directoryURL)

	regCtx, regCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer regCancel()

	regStart := time.Now()
	acct := &acme.Account{
		Contact: []string{"mailto:" + cfg.Email},
	}

	logger.Info("[ACME:INIT] Calling acmeClient.Register...",
		"contact", acct.Contact)

	registeredAcct, err := acmeClient.Register(regCtx, acct, autocert.AcceptTOS)
	regDuration := time.Since(regStart)

	if err != nil {
		// Check if account already exists (this is fine)
		if err == acme.ErrAccountAlreadyExists {
			logger.Info("[ACME:INIT] ACME account already exists (OK)",
				"duration", regDuration,
				"email", cfg.Email)
			// Try to get the existing account
			getAcctCtx, getAcctCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer getAcctCancel()

			getAcctStart := time.Now()
			existingAcct, getErr := acmeClient.GetReg(getAcctCtx, "")
			getAcctDuration := time.Since(getAcctStart)

			if getErr != nil {
				logger.Warn("[ACME:INIT] Could not get existing account (continuing anyway)",
					"duration", getAcctDuration,
					"error", getErr)
			} else {
				logger.Info("[ACME:INIT] Retrieved existing ACME account",
					"duration", getAcctDuration,
					"status", existingAcct.Status,
					"uri", existingAcct.URI,
					"contact", existingAcct.Contact)
			}
		} else {
			// Registration failed with some other error - log prominently but continue
			// This catches network issues, rate limiting, invalid email, etc.
			// autocert will try again when GetCertificate is called
			logger.Error("[ACME:INIT] ACME account registration FAILED (will retry on first request)",
				"duration", regDuration,
				"error", err,
				"error_type", fmt.Sprintf("%T", err),
				"note", "initialization continuing - autocert will retry registration")
		}
	} else {
		logger.Info("[ACME:INIT] ACME account registered successfully",
			"duration", regDuration,
			"status", registeredAcct.Status,
			"uri", registeredAcct.URI,
			"contact", registeredAcct.Contact,
			"orders_url", registeredAcct.OrdersURL)
	}

	// Create autocert manager with explicit Client
	logger.Info("[ACME:INIT] Creating autocert.Manager",
		"email", cfg.Email,
		"has_host_policy", true,
		"has_cache", true,
		"has_client", true)

	provider.manager = &autocert.Manager{
		Cache:      cache,
		Prompt:     autocert.AcceptTOS,
		Email:      cfg.Email,
		HostPolicy: provider.hostPolicy,
		Client:     acmeClient,
	}

	logger.Info("[ACME:INIT] ACME provider initialization complete",
		"staging", cfg.Staging,
		"directory", directoryURL,
		"email", cfg.Email,
		"domains", cfg.Domains)

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
	requestID := fmt.Sprintf("%d", start.UnixNano())

	clientAddr := "unknown"
	if hello.Conn != nil {
		clientAddr = hello.Conn.RemoteAddr().String()
	}

	p.logger.Info("[ACME:GETCERT] TLS certificate requested",
		"request_id", requestID,
		"domain", domain,
		"staging", p.staging,
		"client_addr", clientAddr)

	// Check host policy first for early logging
	p.logger.Debug("[ACME:GETCERT] Checking host policy",
		"request_id", requestID,
		"domain", domain,
		"allowed_domains", p.domains)

	if err := p.hostPolicy(context.Background(), domain); err != nil {
		p.logger.Error("[ACME:GETCERT] domain rejected by host policy",
			"request_id", requestID,
			"domain", domain,
			"error", err,
			"allowed_domains", p.domains,
			"duration", time.Since(start))
		return nil, err
	}
	p.logger.Debug("[ACME:GETCERT] Host policy check passed",
		"request_id", requestID,
		"domain", domain)

	// Check if domain is rate limited
	p.rateLimitMu.RLock()
	retryAfter, isRateLimited := p.rateLimitUntil[domain]
	p.rateLimitMu.RUnlock()

	if isRateLimited && time.Now().Before(retryAfter) {
		waitTime := time.Until(retryAfter)
		p.logger.Warn("[ACME:GETCERT] Domain is rate limited, cannot obtain certificate",
			"request_id", requestID,
			"domain", domain,
			"retry_after", retryAfter.Format(time.RFC3339),
			"wait_time", waitTime.Round(time.Second))
		return nil, fmt.Errorf("rate limited for domain %s, retry after %s (in %s)",
			domain, retryAfter.Format(time.RFC3339), waitTime.Round(time.Second))
	}

	// Log detailed state before calling autocert
	p.logger.Info("[ACME:GETCERT] Calling autocert.Manager.GetCertificate",
		"request_id", requestID,
		"domain", domain,
		"staging", p.staging,
		"acme_directory", p.getDirectoryURL(),
		"email", p.email,
		"manager_has_cache", p.manager.Cache != nil,
		"manager_has_client", p.manager.Client != nil)

	// Use a channel to implement timeout around GetCertificate
	type result struct {
		cert *cryptotls.Certificate
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		p.logger.Info("[ACME:GETCERT] Goroutine started - calling p.manager.GetCertificate",
			"request_id", requestID,
			"domain", domain,
			"goroutine_start", time.Now().Format(time.RFC3339Nano))

		certStart := time.Now()
		cert, err := p.manager.GetCertificate(hello)
		certDuration := time.Since(certStart)

		p.logger.Info("[ACME:GETCERT] p.manager.GetCertificate returned",
			"request_id", requestID,
			"domain", domain,
			"duration", certDuration,
			"has_cert", cert != nil,
			"has_error", err != nil,
			"error", err)

		resultCh <- result{cert, err}
	}()

	// Wait for result with 90 second timeout
	p.logger.Debug("[ACME:GETCERT] Waiting for result (90s timeout)",
		"request_id", requestID,
		"domain", domain)

	select {
	case r := <-resultCh:
		cert, err := r.cert, r.err
		totalDuration := time.Since(start)

		if err != nil {
			// Check if this is a rate limit error and extract retry-after time
			errStr := err.Error()
			if strings.Contains(errStr, "rateLimited") || strings.Contains(errStr, "429") {
				// Parse retry-after time from error message
				// Format: "retry after 2026-01-26 09:41:29 UTC"
				if retryTime := p.parseRetryAfter(errStr); !retryTime.IsZero() {
					p.rateLimitMu.Lock()
					p.rateLimitUntil[domain] = retryTime
					p.rateLimitMu.Unlock()

					p.logger.Error("[ACME:GETCERT] Rate limited by Let's Encrypt",
						"request_id", requestID,
						"domain", domain,
						"duration", totalDuration,
						"retry_after", retryTime.Format(time.RFC3339),
						"wait_time", time.Until(retryTime).Round(time.Second),
						"staging", p.staging)
					return nil, fmt.Errorf("rate limited for domain %s, retry after %s",
						domain, retryTime.Format(time.RFC3339))
				}
			}

			p.logger.Error("[ACME:GETCERT] Certificate acquisition failed",
				"request_id", requestID,
				"domain", domain,
				"duration", totalDuration,
				"staging", p.staging,
				"error", err,
				"error_type", fmt.Sprintf("%T", err))
			return nil, err
		}

		// Log success with certificate details
		if cert != nil && len(cert.Certificate) > 0 {
			if x509Cert, parseErr := x509.ParseCertificate(cert.Certificate[0]); parseErr == nil {
				p.logger.Info("[ACME:GETCERT] Certificate obtained successfully",
					"request_id", requestID,
					"domain", domain,
					"duration", totalDuration,
					"issuer", x509Cert.Issuer.CommonName,
					"issuer_org", x509Cert.Issuer.Organization,
					"subject", x509Cert.Subject.CommonName,
					"not_before", x509Cert.NotBefore,
					"not_after", x509Cert.NotAfter,
					"serial", x509Cert.SerialNumber.String(),
					"dns_names", x509Cert.DNSNames)
			} else {
				p.logger.Info("[ACME:GETCERT] Certificate obtained (parse failed)",
					"request_id", requestID,
					"domain", domain,
					"duration", totalDuration,
					"parse_error", parseErr)
			}
		} else {
			p.logger.Info("[ACME:GETCERT] Certificate obtained (empty cert chain?)",
				"request_id", requestID,
				"domain", domain,
				"duration", totalDuration,
				"cert_is_nil", cert == nil)
		}

		return cert, nil

	case <-time.After(90 * time.Second):
		totalDuration := time.Since(start)
		p.logger.Error("[ACME:GETCERT] Certificate acquisition TIMED OUT",
			"request_id", requestID,
			"domain", domain,
			"duration", totalDuration,
			"staging", p.staging,
			"timeout", "90s",
			"acme_directory", p.getDirectoryURL())
		return nil, fmt.Errorf("certificate acquisition timed out after 90s for domain %s", domain)
	}
}

// hostPolicy checks if a domain is allowed.
func (p *ACMEProvider) hostPolicy(ctx context.Context, host string) error {
	p.mu.RLock()
	domains := p.domains
	p.mu.RUnlock()

	p.logger.Debug("[ACME:HOSTPOLICY] Checking host",
		"host", host,
		"configured_domains", domains,
		"domain_count", len(domains))

	// If no domains configured, allow all
	if len(domains) == 0 {
		p.logger.Debug("[ACME:HOSTPOLICY] No domains configured, allowing all")
		return nil
	}

	// Check if host matches any configured domain
	for _, d := range domains {
		if d == host {
			p.logger.Debug("[ACME:HOSTPOLICY] Exact match found",
				"host", host,
				"matched_domain", d)
			return nil
		}
		// Support wildcard matching
		if len(d) > 1 && d[0] == '*' && d[1] == '.' {
			suffix := d[1:] // .example.com
			if len(host) > len(suffix) && host[len(host)-len(suffix):] == suffix {
				p.logger.Debug("[ACME:HOSTPOLICY] Wildcard match found",
					"host", host,
					"matched_wildcard", d)
				return nil
			}
		}
	}

	p.logger.Warn("[ACME:HOSTPOLICY] Host not in allowed domains",
		"host", host,
		"allowed_domains", domains)
	return fmt.Errorf("host %q not in allowed domains", host)
}

// UpdateDomains updates the list of allowed domains.
func (p *ACMEProvider) UpdateDomains(domains []string) {
	p.logger.Info("[ACME:CONFIG] Updating allowed domains",
		"old_domains", p.domains,
		"new_domains", domains)
	p.mu.Lock()
	p.domains = domains
	p.mu.Unlock()
}

// GetCertificate retrieves or obtains a certificate for a domain.
// This is the main method called by tls.Config.GetCertificate.
func (p *ACMEProvider) GetCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	p.logger.Debug("[ACME:GETCERT:INTERNAL] GetCertificate called",
		"domain", domain)

	// Try to get from database first
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err == nil && cert.IsActive() && !cert.NeedsRenewal(p.renewalDays) {
		p.logger.Debug("[ACME:GETCERT:INTERNAL] Found valid certificate in database",
			"domain", domain,
			"expires", cert.ExpiresAt)
		return cert, nil
	}

	p.logger.Info("[ACME:GETCERT:INTERNAL] Need to obtain/renew certificate",
		"domain", domain,
		"db_error", err,
		"cert_active", cert.IsActive(),
		"needs_renewal", cert.NeedsRenewal(p.renewalDays))

	// Need to obtain/renew certificate
	return p.ObtainCertificate(ctx, domain)
}

// ObtainCertificate obtains a new certificate for a domain via ACME.
func (p *ACMEProvider) ObtainCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	p.logger.Info("[ACME:OBTAIN] Starting certificate acquisition",
		"domain", domain)

	// Check host policy first
	if err := p.hostPolicy(ctx, domain); err != nil {
		p.logger.Error("[ACME:OBTAIN] Host policy check failed",
			"domain", domain,
			"error", err)
		return domaintls.Certificate{}, err
	}

	// Use autocert manager to get the certificate
	hello := &cryptotls.ClientHelloInfo{
		ServerName: domain,
	}

	p.logger.Info("[ACME:OBTAIN] Calling autocert.Manager.GetCertificate",
		"domain", domain)

	tlsCert, err := p.manager.GetCertificate(hello)
	if err != nil {
		p.logger.Error("[ACME:OBTAIN] Failed to get certificate from ACME",
			"domain", domain,
			"error", err)
		return domaintls.Certificate{}, fmt.Errorf("get certificate from ACME: %w", err)
	}

	// Parse the certificate to extract metadata
	if len(tlsCert.Certificate) == 0 {
		p.logger.Error("[ACME:OBTAIN] No certificate returned from ACME",
			"domain", domain)
		return domaintls.Certificate{}, errors.New("no certificate returned from ACME")
	}

	_, err = x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		p.logger.Error("[ACME:OBTAIN] Failed to parse certificate",
			"domain", domain,
			"error", err)
		return domaintls.Certificate{}, fmt.Errorf("parse certificate: %w", err)
	}

	p.logger.Info("[ACME:OBTAIN] Certificate obtained, fetching from database",
		"domain", domain)

	// Get from database (should have been stored by cache)
	return p.certStore.GetByDomain(ctx, domain)
}

// RenewCertificate forces renewal of a certificate.
func (p *ACMEProvider) RenewCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	p.logger.Info("[ACME:RENEW] Starting certificate renewal",
		"domain", domain)

	// Delete from cache to force re-fetch
	if err := p.cache.Delete(ctx, domain); err != nil {
		p.logger.Warn("[ACME:RENEW] Failed to delete from cache (continuing anyway)",
			"domain", domain,
			"error", err)
	}

	// Clear memory cache
	p.cache.ClearMemoryCache()
	p.logger.Debug("[ACME:RENEW] Memory cache cleared")

	// Obtain new certificate
	return p.ObtainCertificate(ctx, domain)
}

// RevokeCertificate revokes a certificate.
func (p *ACMEProvider) RevokeCertificate(ctx context.Context, domain string, reason string) error {
	p.logger.Info("[ACME:REVOKE] Starting certificate revocation",
		"domain", domain,
		"reason", reason)

	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err != nil {
		p.logger.Error("[ACME:REVOKE] Failed to get certificate from store",
			"domain", domain,
			"error", err)
		return fmt.Errorf("get certificate: %w", err)
	}

	// Parse the certificate
	block, _ := pem.Decode(cert.CertPEM)
	if block == nil {
		p.logger.Error("[ACME:REVOKE] Failed to decode certificate PEM",
			"domain", domain)
		return errors.New("failed to decode certificate PEM")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		p.logger.Error("[ACME:REVOKE] Failed to parse certificate",
			"domain", domain,
			"error", err)
		return fmt.Errorf("parse certificate: %w", err)
	}

	// Revoke via ACME
	p.logger.Info("[ACME:REVOKE] Calling ACME revoke endpoint",
		"domain", domain,
		"serial", x509Cert.SerialNumber.String())

	err = p.acmeClient.RevokeCert(ctx, nil, x509Cert.Raw, acme.CRLReasonUnspecified)
	if err != nil {
		p.logger.Error("[ACME:REVOKE] ACME revocation failed",
			"domain", domain,
			"error", err)
		return fmt.Errorf("revoke certificate via ACME: %w", err)
	}

	// Update status in database
	now := time.Now().UTC()
	cert.Status = domaintls.StatusRevoked
	cert.RevokedAt = &now
	cert.RevokeReason = reason

	if err := p.certStore.Update(ctx, cert); err != nil {
		p.logger.Error("[ACME:REVOKE] Failed to update certificate status in database",
			"domain", domain,
			"error", err)
		return fmt.Errorf("update certificate status: %w", err)
	}

	// Remove from cache
	p.cache.Delete(ctx, domain)

	p.logger.Info("[ACME:REVOKE] Certificate revoked successfully",
		"domain", domain)
	return nil
}

// CheckRenewal checks if a certificate needs renewal.
func (p *ACMEProvider) CheckRenewal(ctx context.Context, domain string, renewalDays int) (bool, error) {
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			p.logger.Debug("[ACME:RENEWAL] No certificate found, needs obtaining",
				"domain", domain)
			return true, nil
		}
		return false, fmt.Errorf("get certificate: %w", err)
	}

	needsRenewal := cert.NeedsRenewal(renewalDays)
	p.logger.Debug("[ACME:RENEWAL] Renewal check complete",
		"domain", domain,
		"needs_renewal", needsRenewal,
		"expires", cert.ExpiresAt,
		"renewal_days", renewalDays)

	return needsRenewal, nil
}

// GetManager returns the autocert manager for use in HTTP handlers.
// This can be used to handle ACME HTTP-01 challenges.
func (p *ACMEProvider) GetManager() *autocert.Manager {
	return p.manager
}

// parseRetryAfter extracts the retry-after time from a rate limit error message.
// Looks for patterns like "retry after 2026-01-26 09:41:29 UTC" in the error string.
func (p *ACMEProvider) parseRetryAfter(errStr string) time.Time {
	// Pattern: "retry after YYYY-MM-DD HH:MM:SS UTC"
	re := regexp.MustCompile(`retry after (\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) UTC`)
	matches := re.FindStringSubmatch(errStr)
	if len(matches) >= 2 {
		t, err := time.Parse("2006-01-02 15:04:05", matches[1])
		if err == nil {
			return t.UTC()
		}
		p.logger.Warn("[ACME:RATELIMIT] Failed to parse retry-after time",
			"matched", matches[1],
			"error", err)
	}

	// If we can't parse the exact time, set a default wait of 1 hour
	// This is safer than hammering the ACME server
	p.logger.Warn("[ACME:RATELIMIT] Could not parse retry-after time, defaulting to 1 hour",
		"error_string_preview", errStr[:min(100, len(errStr))])
	return time.Now().Add(1 * time.Hour)
}

// ClearRateLimit clears the rate limit for a domain (useful after waiting).
func (p *ACMEProvider) ClearRateLimit(domain string) {
	p.rateLimitMu.Lock()
	delete(p.rateLimitUntil, domain)
	p.rateLimitMu.Unlock()
	p.logger.Info("[ACME:RATELIMIT] Rate limit cleared for domain", "domain", domain)
}

// GetRateLimitInfo returns rate limit information for a domain.
func (p *ACMEProvider) GetRateLimitInfo(domain string) (time.Time, bool) {
	p.rateLimitMu.RLock()
	defer p.rateLimitMu.RUnlock()
	retryAfter, exists := p.rateLimitUntil[domain]
	if !exists || time.Now().After(retryAfter) {
		return time.Time{}, false
	}
	return retryAfter, true
}

// Ensure interface compliance.
var _ ports.TLSProvider = (*ACMEProvider)(nil)

// ClientHelloInfo is a type alias for compatibility.
type ClientHelloInfo = cryptotls.ClientHelloInfo
