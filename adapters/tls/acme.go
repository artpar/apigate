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
)

const (
	// LetsEncrypt production directory
	letsEncryptProduction = "https://acme-v02.api.letsencrypt.org/directory"
	// LetsEncrypt staging directory (for testing)
	letsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"

	// Per-step timeout for ACME operations
	acmeStepTimeout = 30 * time.Second

	// Challenge types
	challengeTLSALPN01 = "tls-alpn-01"
	challengeHTTP01    = "http-01"
)

// ACMEErrorType classifies ACME errors for handling decisions.
type ACMEErrorType int

const (
	// ErrorRetryable indicates network issues or temporary failures - can retry.
	ErrorRetryable ACMEErrorType = iota
	// ErrorRateLimited indicates 429 - fast-fail, don't retry until retry-after.
	ErrorRateLimited
	// ErrorInvalid indicates 4xx - bad request, don't retry.
	ErrorInvalid
	// ErrorFatal indicates unrecoverable errors.
	ErrorFatal
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

// ACMEProvider implements TLS certificate provisioning via direct ACME client.
// This replaces autocert.Manager with direct control over each ACME step.
type ACMEProvider struct {
	// Dependencies
	certStore  ports.CertificateStore
	cacheStore ports.ACMECacheStore
	logger     *slog.Logger

	// ACME configuration
	client      *acme.Client
	email       string
	staging     bool
	renewalDays int

	// Domain management
	mu      sync.RWMutex
	domains []string

	// Account key
	accountKey crypto.Signer

	// Challenge handling
	challengeMu  sync.RWMutex
	tlsAlpnCerts map[string]*cryptotls.Certificate // domain -> TLS-ALPN-01 challenge cert
	http01Tokens map[string]string                 // token -> key authorization

	// Rate limit tracking (fast-fail)
	rateLimitMu    sync.RWMutex
	rateLimitUntil map[string]time.Time

	// In-memory cert cache for performance
	certCacheMu sync.RWMutex
	certCache   map[string]*cryptotls.Certificate
}

// ACMEConfig holds configuration for ACME provider.
type ACMEConfig struct {
	Email       string
	Staging     bool     // Use staging server for testing
	Domains     []string // Domains to obtain certificates for
	RenewalDays int      // Days before expiry to renew (default: 30)
}

// NewACMEProvider creates a new direct ACME TLS provider.
func NewACMEProvider(certStore ports.CertificateStore, cfg ACMEConfig) (*ACMEProvider, error) {
	logger := slog.Default()

	logger.Info("[ACME:INIT] Starting direct ACME provider initialization",
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
		"step_timeout", acmeStepTimeout,
		"dial_timeout", "10s",
		"force_ipv4", true)

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	loggingTransport := &loggingRoundTripper{
		wrapped: &http.Transport{
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

	acmeClient := &acme.Client{
		DirectoryURL: directoryURL,
		Key:          accountKey,
		HTTPClient:   acmeHTTPClient,
	}

	provider := &ACMEProvider{
		certStore:      certStore,
		cacheStore:     cacheStore,
		email:          cfg.Email,
		staging:        cfg.Staging,
		renewalDays:    renewalDays,
		domains:        cfg.Domains,
		accountKey:     accountKey,
		logger:         logger,
		client:         acmeClient,
		tlsAlpnCerts:   make(map[string]*cryptotls.Certificate),
		http01Tokens:   make(map[string]string),
		rateLimitUntil: make(map[string]time.Time),
		certCache:      make(map[string]*cryptotls.Certificate),
	}

	// Pre-fetch the ACME directory
	logger.Info("[ACME:INIT] Pre-fetching ACME directory...",
		"url", directoryURL)

	ctx, cancel := context.WithTimeout(context.Background(), acmeStepTimeout)
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
		"authz_url", dir.AuthzURL,
		"order_url", dir.OrderURL)

	// Pre-register the ACME account
	logger.Info("[ACME:INIT] Pre-registering ACME account...",
		"email", cfg.Email)

	regCtx, regCancel := context.WithTimeout(context.Background(), acmeStepTimeout)
	defer regCancel()

	regStart := time.Now()
	acct := &acme.Account{
		Contact: []string{"mailto:" + cfg.Email},
	}

	registeredAcct, err := acmeClient.Register(regCtx, acct, func(tosURL string) bool {
		logger.Info("[ACME:INIT] Accepting Terms of Service", "url", tosURL)
		return true
	})
	regDuration := time.Since(regStart)

	if err != nil {
		if err == acme.ErrAccountAlreadyExists {
			logger.Info("[ACME:INIT] ACME account already exists (OK)",
				"duration", regDuration)
		} else {
			logger.Error("[ACME:INIT] ACME account registration FAILED",
				"duration", regDuration,
				"error", err,
				"note", "will retry on first certificate request")
		}
	} else {
		logger.Info("[ACME:INIT] ACME account registered successfully",
			"duration", regDuration,
			"status", registeredAcct.Status,
			"uri", registeredAcct.URI)
	}

	logger.Info("[ACME:INIT] Direct ACME provider initialization complete",
		"staging", cfg.Staging,
		"directory", directoryURL)

	return provider, nil
}

// Name returns the provider name.
func (p *ACMEProvider) Name() string {
	return "acme"
}

// SetLogger sets a custom logger for the ACME provider.
func (p *ACMEProvider) SetLogger(logger *slog.Logger) {
	p.logger = logger
}

// getDirectoryURL returns the ACME directory URL.
func (p *ACMEProvider) getDirectoryURL() string {
	if p.staging {
		return letsEncryptStaging
	}
	return letsEncryptProduction
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

// hostPolicy checks if a domain is allowed.
func (p *ACMEProvider) hostPolicy(_ context.Context, host string) error {
	p.mu.RLock()
	domains := p.domains
	p.mu.RUnlock()

	// If no domains configured, allow all
	if len(domains) == 0 {
		return nil
	}

	for _, d := range domains {
		if d == host {
			return nil
		}
		// Support wildcard matching
		if len(d) > 1 && d[0] == '*' && d[1] == '.' {
			suffix := d[1:] // .example.com
			if len(host) > len(suffix) && host[len(host)-len(suffix):] == suffix {
				return nil
			}
		}
	}

	return fmt.Errorf("host %q not in allowed domains", host)
}

// classifyError classifies an ACME error for handling decisions.
func (p *ACMEProvider) classifyError(err error) ACMEErrorType {
	if err == nil {
		return ErrorRetryable
	}

	// Check for rate limit
	if _, ok := acme.RateLimit(err); ok {
		return ErrorRateLimited
	}

	// Check for ACME error types
	var acmeErr *acme.Error
	if errors.As(err, &acmeErr) {
		switch acmeErr.StatusCode {
		case 429:
			return ErrorRateLimited
		case 400, 403, 404:
			return ErrorInvalid
		case 500, 502, 503:
			return ErrorRetryable
		}
	}

	// Network errors are retryable
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrorRetryable
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return ErrorRetryable
	}

	return ErrorFatal
}

// recordRateLimit records a rate limit for a domain.
func (p *ACMEProvider) recordRateLimit(domain string, retryAfter time.Time) {
	p.rateLimitMu.Lock()
	p.rateLimitUntil[domain] = retryAfter
	p.rateLimitMu.Unlock()

	p.logger.Warn("[ACME:RATELIMIT] Rate limit recorded",
		"domain", domain,
		"retry_after", retryAfter.Format(time.RFC3339),
		"wait_time", time.Until(retryAfter).Round(time.Second))
}

// isRateLimited checks if a domain is currently rate limited.
func (p *ACMEProvider) isRateLimited(domain string) bool {
	p.rateLimitMu.RLock()
	retryAfter, exists := p.rateLimitUntil[domain]
	p.rateLimitMu.RUnlock()

	if !exists {
		return false
	}
	return time.Now().Before(retryAfter)
}

// getRateLimitExpiry returns the rate limit expiry time for a domain.
func (p *ACMEProvider) getRateLimitExpiry(domain string) time.Time {
	p.rateLimitMu.RLock()
	defer p.rateLimitMu.RUnlock()
	return p.rateLimitUntil[domain]
}

// ClearRateLimit clears the rate limit for a domain.
func (p *ACMEProvider) ClearRateLimit(domain string) {
	p.rateLimitMu.Lock()
	delete(p.rateLimitUntil, domain)
	p.rateLimitMu.Unlock()
	p.logger.Info("[ACME:RATELIMIT] Rate limit cleared", "domain", domain)
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

// getCachedCert returns a certificate from memory cache.
func (p *ACMEProvider) getCachedCert(domain string) *cryptotls.Certificate {
	p.certCacheMu.RLock()
	defer p.certCacheMu.RUnlock()
	return p.certCache[domain]
}

// setCachedCert stores a certificate in memory cache.
func (p *ACMEProvider) setCachedCert(domain string, cert *cryptotls.Certificate) {
	p.certCacheMu.Lock()
	p.certCache[domain] = cert
	p.certCacheMu.Unlock()
}

// loadFromDatabase loads a certificate from the database.
func (p *ACMEProvider) loadFromDatabase(ctx context.Context, domain string) (*cryptotls.Certificate, error) {
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err != nil {
		return nil, err
	}

	if !cert.IsActive() {
		return nil, fmt.Errorf("certificate not active: status=%s", cert.Status)
	}

	if cert.NeedsRenewal(p.renewalDays) {
		p.logger.Info("[ACME:CACHE] Certificate needs renewal",
			"domain", domain,
			"expires", cert.ExpiresAt,
			"renewal_days", p.renewalDays)
		// Still return it for now, but caller should trigger renewal
	}

	// Convert to tls.Certificate
	tlsCert, err := cryptotls.X509KeyPair(cert.CertPEM, cert.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	// Cache in memory
	p.setCachedCert(domain, &tlsCert)

	return &tlsCert, nil
}

// GetTLSALPNCert returns a TLS-ALPN-01 challenge certificate if one exists for the domain.
func (p *ACMEProvider) GetTLSALPNCert(hello *cryptotls.ClientHelloInfo) (*cryptotls.Certificate, bool) {
	// Check if this is an ACME TLS-ALPN-01 challenge
	for _, proto := range hello.SupportedProtos {
		if proto == "acme-tls/1" {
			p.challengeMu.RLock()
			cert, exists := p.tlsAlpnCerts[hello.ServerName]
			p.challengeMu.RUnlock()

			if exists {
				p.logger.Info("[ACME:CHALLENGE] Serving TLS-ALPN-01 challenge cert",
					"domain", hello.ServerName)
				return cert, true
			}
		}
	}
	return nil, false
}

// prepareTLSALPN01Challenge prepares a TLS-ALPN-01 challenge.
func (p *ACMEProvider) prepareTLSALPN01Challenge(domain, token string) error {
	p.logger.Info("[ACME:CHALLENGE] Preparing TLS-ALPN-01 challenge",
		"domain", domain)

	cert, err := p.client.TLSALPN01ChallengeCert(token, domain)
	if err != nil {
		return fmt.Errorf("generate TLS-ALPN-01 cert: %w", err)
	}

	p.challengeMu.Lock()
	p.tlsAlpnCerts[domain] = &cert
	p.challengeMu.Unlock()

	p.logger.Info("[ACME:CHALLENGE] TLS-ALPN-01 challenge prepared",
		"domain", domain)
	return nil
}

// cleanupTLSALPN01Challenge removes a TLS-ALPN-01 challenge certificate.
func (p *ACMEProvider) cleanupTLSALPN01Challenge(domain string) {
	p.challengeMu.Lock()
	delete(p.tlsAlpnCerts, domain)
	p.challengeMu.Unlock()
	p.logger.Debug("[ACME:CHALLENGE] TLS-ALPN-01 challenge cleaned up", "domain", domain)
}

// prepareHTTP01Challenge prepares an HTTP-01 challenge.
func (p *ACMEProvider) prepareHTTP01Challenge(token string) (string, error) {
	keyAuth, err := p.client.HTTP01ChallengeResponse(token)
	if err != nil {
		return "", fmt.Errorf("generate HTTP-01 response: %w", err)
	}

	p.challengeMu.Lock()
	p.http01Tokens[token] = keyAuth
	p.challengeMu.Unlock()

	p.logger.Info("[ACME:CHALLENGE] HTTP-01 challenge prepared",
		"token", token[:min(8, len(token))]+"...")
	return keyAuth, nil
}

// cleanupHTTP01Challenge removes an HTTP-01 challenge token.
func (p *ACMEProvider) cleanupHTTP01Challenge(token string) {
	p.challengeMu.Lock()
	delete(p.http01Tokens, token)
	p.challengeMu.Unlock()
}

// ServeHTTP01Challenge handles HTTP-01 challenge requests.
// Mount this at /.well-known/acme-challenge/
func (p *ACMEProvider) ServeHTTP01Challenge(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.URL.Path, "/.well-known/acme-challenge/")
	if token == "" {
		http.NotFound(w, r)
		return
	}

	p.challengeMu.RLock()
	keyAuth, exists := p.http01Tokens[token]
	p.challengeMu.RUnlock()

	if !exists {
		p.logger.Warn("[ACME:CHALLENGE] HTTP-01 token not found",
			"token", token[:min(8, len(token))]+"...")
		http.NotFound(w, r)
		return
	}

	p.logger.Info("[ACME:CHALLENGE] Serving HTTP-01 challenge",
		"token", token[:min(8, len(token))]+"...")

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(keyAuth))
}

// HTTPHandler returns an HTTP handler that serves ACME HTTP-01 challenges.
// This is compatible with autocert.Manager.HTTPHandler for easy migration.
// If fallback is nil, non-challenge requests return 404.
func (p *ACMEProvider) HTTPHandler(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			p.ServeHTTP01Challenge(w, r)
			return
		}
		if fallback != nil {
			fallback.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

// selectChallenge selects the best challenge type from available challenges.
// Prefers TLS-ALPN-01 over HTTP-01.
func (p *ACMEProvider) selectChallenge(challenges []*acme.Challenge) *acme.Challenge {
	var tlsAlpn, http01 *acme.Challenge

	for _, ch := range challenges {
		switch ch.Type {
		case challengeTLSALPN01:
			tlsAlpn = ch
		case challengeHTTP01:
			http01 = ch
		}
	}

	// Prefer TLS-ALPN-01
	if tlsAlpn != nil {
		return tlsAlpn
	}
	return http01
}

// obtainCertificateDirect obtains a certificate using direct ACME flow.
// This is the main method that implements the 5-step ACME flow with explicit caching.
func (p *ACMEProvider) obtainCertificateDirect(ctx context.Context, domain string) (*cryptotls.Certificate, error) {
	flowStart := time.Now()
	p.logger.Info("[ACME:FLOW:START] Starting direct ACME certificate acquisition",
		"domain", domain,
		"staging", p.staging)

	// Check host policy
	if err := p.hostPolicy(ctx, domain); err != nil {
		return nil, err
	}

	// Step 1: Create Order
	p.logger.Info("[ACME:ORDER] Creating order...",
		"domain", domain)

	orderCtx, orderCancel := context.WithTimeout(ctx, acmeStepTimeout)
	defer orderCancel()

	orderStart := time.Now()
	order, err := p.client.AuthorizeOrder(orderCtx, acme.DomainIDs(domain))
	orderDuration := time.Since(orderStart)

	if err != nil {
		errType := p.classifyError(err)
		if errType == ErrorRateLimited {
			if retryAfter, ok := acme.RateLimit(err); ok && retryAfter > 0 {
				p.recordRateLimit(domain, time.Now().Add(retryAfter))
			} else {
				p.recordRateLimit(domain, time.Now().Add(1*time.Hour))
			}
		}
		p.logger.Error("[ACME:ORDER:FAIL] Failed to create order",
			"domain", domain,
			"duration", orderDuration,
			"error", err,
			"error_type", errType)
		return nil, fmt.Errorf("create order: %w", err)
	}

	p.logger.Info("[ACME:ORDER:SUCCESS] Order created",
		"domain", domain,
		"duration", orderDuration,
		"order_url", order.URI,
		"status", order.Status,
		"authz_count", len(order.AuthzURLs))

	// Step 2: Get Authorization and solve challenges
	for _, authzURL := range order.AuthzURLs {
		p.logger.Info("[ACME:AUTHZ] Processing authorization...",
			"domain", domain,
			"authz_url", authzURL)

		authzCtx, authzCancel := context.WithTimeout(ctx, acmeStepTimeout)

		authzStart := time.Now()
		authz, err := p.client.GetAuthorization(authzCtx, authzURL)
		authzCancel()
		authzDuration := time.Since(authzStart)

		if err != nil {
			p.logger.Error("[ACME:AUTHZ:FAIL] Failed to get authorization",
				"domain", domain,
				"authz_url", authzURL,
				"duration", authzDuration,
				"error", err)
			return nil, fmt.Errorf("get authorization: %w", err)
		}

		p.logger.Info("[ACME:AUTHZ:SUCCESS] Authorization retrieved",
			"domain", domain,
			"duration", authzDuration,
			"status", authz.Status,
			"identifier", authz.Identifier.Value,
			"challenge_count", len(authz.Challenges))

		// If already valid, skip challenge
		if authz.Status == acme.StatusValid {
			p.logger.Info("[ACME:AUTHZ] Authorization already valid, skipping challenge",
				"domain", domain)
			continue
		}

		// Step 3: Solve Challenge
		challenge := p.selectChallenge(authz.Challenges)
		if challenge == nil {
			p.logger.Error("[ACME:CHALLENGE:FAIL] No supported challenge type found",
				"domain", domain,
				"available", func() []string {
					types := make([]string, len(authz.Challenges))
					for i, c := range authz.Challenges {
						types[i] = c.Type
					}
					return types
				}())
			return nil, fmt.Errorf("no supported challenge type for domain %s", domain)
		}

		p.logger.Info("[ACME:CHALLENGE] Preparing challenge...",
			"domain", domain,
			"type", challenge.Type,
			"token", challenge.Token[:min(8, len(challenge.Token))]+"...")

		// Prepare the challenge
		switch challenge.Type {
		case challengeTLSALPN01:
			if err := p.prepareTLSALPN01Challenge(domain, challenge.Token); err != nil {
				return nil, fmt.Errorf("prepare TLS-ALPN-01: %w", err)
			}
			defer p.cleanupTLSALPN01Challenge(domain)
		case challengeHTTP01:
			if _, err := p.prepareHTTP01Challenge(challenge.Token); err != nil {
				return nil, fmt.Errorf("prepare HTTP-01: %w", err)
			}
			defer p.cleanupHTTP01Challenge(challenge.Token)
		}

		// Accept the challenge
		p.logger.Info("[ACME:CHALLENGE] Accepting challenge...",
			"domain", domain,
			"type", challenge.Type)

		acceptCtx, acceptCancel := context.WithTimeout(ctx, acmeStepTimeout)
		acceptStart := time.Now()
		_, err = p.client.Accept(acceptCtx, challenge)
		acceptCancel()
		acceptDuration := time.Since(acceptStart)

		if err != nil {
			p.logger.Error("[ACME:CHALLENGE:FAIL] Failed to accept challenge",
				"domain", domain,
				"type", challenge.Type,
				"duration", acceptDuration,
				"error", err)
			return nil, fmt.Errorf("accept challenge: %w", err)
		}

		p.logger.Info("[ACME:CHALLENGE] Challenge accepted, waiting for validation...",
			"domain", domain,
			"type", challenge.Type,
			"duration", acceptDuration)

		// Wait for authorization to become valid
		waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Minute)
		waitStart := time.Now()
		authz, err = p.client.WaitAuthorization(waitCtx, authzURL)
		waitCancel()
		waitDuration := time.Since(waitStart)

		if err != nil {
			p.logger.Error("[ACME:CHALLENGE:FAIL] Authorization validation failed",
				"domain", domain,
				"duration", waitDuration,
				"error", err)
			return nil, fmt.Errorf("wait authorization: %w", err)
		}

		p.logger.Info("[ACME:CHALLENGE:SUCCESS] Authorization validated",
			"domain", domain,
			"duration", waitDuration,
			"status", authz.Status)
	}

	// Step 4: Finalize Order
	p.logger.Info("[ACME:FINALIZE] Finalizing order...",
		"domain", domain)

	// Generate certificate key
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate certificate key: %w", err)
	}

	// Create CSR
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		DNSNames: []string{domain},
	}, certKey)
	if err != nil {
		return nil, fmt.Errorf("create CSR: %w", err)
	}

	finalizeCtx, finalizeCancel := context.WithTimeout(ctx, acmeStepTimeout)
	defer finalizeCancel()

	finalizeStart := time.Now()
	certDER, certURL, err := p.client.CreateOrderCert(finalizeCtx, order.FinalizeURL, csr, true)
	finalizeDuration := time.Since(finalizeStart)

	if err != nil {
		errType := p.classifyError(err)
		if errType == ErrorRateLimited {
			if retryAfter, ok := acme.RateLimit(err); ok && retryAfter > 0 {
				p.recordRateLimit(domain, time.Now().Add(retryAfter))
			} else {
				p.recordRateLimit(domain, time.Now().Add(1*time.Hour))
			}
		}
		p.logger.Error("[ACME:FINALIZE:FAIL] Failed to finalize order",
			"domain", domain,
			"duration", finalizeDuration,
			"error", err,
			"error_type", errType)
		return nil, fmt.Errorf("finalize order: %w", err)
	}

	p.logger.Info("[ACME:FINALIZE:SUCCESS] Order finalized, certificate issued",
		"domain", domain,
		"duration", finalizeDuration,
		"cert_url", certURL,
		"cert_count", len(certDER))

	// Step 5: Cache Certificate IMMEDIATELY
	tlsCert, err := p.cacheCertificate(ctx, domain, certDER, certKey)
	if err != nil {
		// This is CRITICAL - certificate was issued but not cached
		p.logger.Error("[ACME:CACHE:FAIL] CRITICAL - Certificate issued but NOT cached!",
			"domain", domain,
			"error", err,
			"cert_url", certURL)
		// Still return the cert even if caching failed
	}

	totalDuration := time.Since(flowStart)
	p.logger.Info("[ACME:FLOW:COMPLETE] Certificate acquisition complete",
		"domain", domain,
		"total_duration", totalDuration,
		"staging", p.staging)

	return tlsCert, nil
}

// cacheCertificate stores the certificate in the database and returns a tls.Certificate.
func (p *ACMEProvider) cacheCertificate(ctx context.Context, domain string, certDER [][]byte, key crypto.Signer) (*cryptotls.Certificate, error) {
	p.logger.Info("[ACME:CACHE:START] Caching certificate IMMEDIATELY after issuance",
		"domain", domain,
		"chain_length", len(certDER))

	if len(certDER) == 0 {
		return nil, errors.New("no certificate data")
	}

	// Parse the leaf certificate
	leafCert, err := x509.ParseCertificate(certDER[0])
	if err != nil {
		return nil, fmt.Errorf("parse leaf certificate: %w", err)
	}

	// Encode certificate PEM
	var certPEM []byte
	certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER[0],
	})...)

	// Encode chain PEM
	var chainPEM []byte
	for i := 1; i < len(certDER); i++ {
		chainPEM = append(chainPEM, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certDER[i],
		})...)
	}

	// Encode key PEM
	var keyPEM []byte
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		keyBytes, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("marshal EC key: %w", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		})
	default:
		keyBytes, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("marshal key: %w", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		})
	}

	now := time.Now().UTC()

	// Check if certificate already exists
	existing, err := p.certStore.GetByDomain(ctx, domain)
	if err == nil && existing.ID != "" {
		// Update existing
		p.logger.Info("[ACME:CACHE] Updating existing certificate record",
			"domain", domain,
			"existing_id", existing.ID)

		existing.CertPEM = certPEM
		existing.KeyPEM = keyPEM
		existing.ChainPEM = chainPEM
		existing.IssuedAt = leafCert.NotBefore
		existing.ExpiresAt = leafCert.NotAfter
		existing.Issuer = leafCert.Issuer.CommonName
		existing.SerialNumber = leafCert.SerialNumber.String()
		existing.Status = domaintls.StatusActive
		existing.UpdatedAt = now

		if err := p.certStore.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("update certificate in database: %w", err)
		}

		p.logger.Info("[ACME:CACHE:SUCCESS] Certificate updated in database",
			"domain", domain,
			"cert_id", existing.ID,
			"issuer", leafCert.Issuer.CommonName,
			"expires", leafCert.NotAfter)
	} else {
		// Create new
		cert := domaintls.Certificate{
			ID:           domaintls.GenerateCertificateID(),
			Domain:       domain,
			CertPEM:      certPEM,
			ChainPEM:     chainPEM,
			KeyPEM:       keyPEM,
			IssuedAt:     leafCert.NotBefore,
			ExpiresAt:    leafCert.NotAfter,
			Issuer:       leafCert.Issuer.CommonName,
			SerialNumber: leafCert.SerialNumber.String(),
			Status:       domaintls.StatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := p.certStore.Create(ctx, cert); err != nil {
			return nil, fmt.Errorf("store certificate in database: %w", err)
		}

		p.logger.Info("[ACME:CACHE:SUCCESS] New certificate stored in database",
			"domain", domain,
			"cert_id", cert.ID,
			"issuer", leafCert.Issuer.CommonName,
			"expires", leafCert.NotAfter,
			"serial", leafCert.SerialNumber.String())
	}

	// Build tls.Certificate
	tlsCert := &cryptotls.Certificate{
		Certificate: certDER,
		PrivateKey:  key,
		Leaf:        leafCert,
	}

	// Cache in memory
	p.setCachedCert(domain, tlsCert)

	return tlsCert, nil
}

// GetCertificateWithLogging is the main TLS GetCertificate function with full logging.
// Use this in tls.Config.GetCertificate.
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

	// 1. Check for TLS-ALPN-01 challenge
	if cert, ok := p.GetTLSALPNCert(hello); ok {
		p.logger.Info("[ACME:GETCERT] Serving TLS-ALPN-01 challenge cert",
			"request_id", requestID,
			"domain", domain,
			"duration", time.Since(start))
		return cert, nil
	}

	// 2. Check host policy
	if err := p.hostPolicy(context.Background(), domain); err != nil {
		p.logger.Error("[ACME:GETCERT] Domain rejected by host policy",
			"request_id", requestID,
			"domain", domain,
			"error", err,
			"duration", time.Since(start))
		return nil, err
	}

	// 3. Check rate limit (fast-fail)
	if p.isRateLimited(domain) {
		expiry := p.getRateLimitExpiry(domain)
		p.logger.Warn("[ACME:GETCERT] Domain is rate limited, fast-fail",
			"request_id", requestID,
			"domain", domain,
			"retry_after", expiry.Format(time.RFC3339),
			"wait_time", time.Until(expiry).Round(time.Second))
		return nil, fmt.Errorf("rate limited for domain %s, retry after %s", domain, expiry.Format(time.RFC3339))
	}

	// 4. Check memory cache
	if cert := p.getCachedCert(domain); cert != nil {
		p.logger.Info("[ACME:GETCERT] Found in memory cache",
			"request_id", requestID,
			"domain", domain,
			"duration", time.Since(start))
		return cert, nil
	}

	// 5. Check database
	ctx := context.Background()
	if cert, err := p.loadFromDatabase(ctx, domain); err == nil {
		p.logger.Info("[ACME:GETCERT] Loaded from database",
			"request_id", requestID,
			"domain", domain,
			"duration", time.Since(start))
		return cert, nil
	}

	// 6. Obtain new certificate via direct ACME flow
	p.logger.Info("[ACME:GETCERT] Certificate not found, obtaining via ACME...",
		"request_id", requestID,
		"domain", domain)

	cert, err := p.obtainCertificateDirect(ctx, domain)
	totalDuration := time.Since(start)

	if err != nil {
		p.logger.Error("[ACME:GETCERT:FAIL] Failed to obtain certificate",
			"request_id", requestID,
			"domain", domain,
			"duration", totalDuration,
			"error", err)
		return nil, err
	}

	p.logger.Info("[ACME:GETCERT:SUCCESS] Certificate obtained",
		"request_id", requestID,
		"domain", domain,
		"duration", totalDuration)

	return cert, nil
}

// GetCertificate retrieves or obtains a certificate for a domain.
// This is the main method called by tls.Config.GetCertificate (via interface).
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

	// Obtain via direct ACME flow
	_, err := p.obtainCertificateDirect(ctx, domain)
	if err != nil {
		p.logger.Error("[ACME:OBTAIN] Failed to obtain certificate",
			"domain", domain,
			"error", err)
		return domaintls.Certificate{}, err
	}

	// Get from database (should have been stored by cacheCertificate)
	return p.certStore.GetByDomain(ctx, domain)
}

// RenewCertificate forces renewal of a certificate.
func (p *ACMEProvider) RenewCertificate(ctx context.Context, domain string) (domaintls.Certificate, error) {
	p.logger.Info("[ACME:RENEW] Starting certificate renewal",
		"domain", domain)

	// Clear memory cache to force re-fetch
	p.certCacheMu.Lock()
	delete(p.certCache, domain)
	p.certCacheMu.Unlock()

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
	revokeCtx, cancel := context.WithTimeout(ctx, acmeStepTimeout)
	defer cancel()

	err = p.client.RevokeCert(revokeCtx, nil, x509Cert.Raw, acme.CRLReasonUnspecified)
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
		return fmt.Errorf("update certificate status: %w", err)
	}

	// Remove from memory cache
	p.certCacheMu.Lock()
	delete(p.certCache, domain)
	p.certCacheMu.Unlock()

	p.logger.Info("[ACME:REVOKE] Certificate revoked successfully",
		"domain", domain)
	return nil
}

// CheckRenewal checks if a certificate needs renewal.
func (p *ACMEProvider) CheckRenewal(ctx context.Context, domain string, renewalDays int) (bool, error) {
	cert, err := p.certStore.GetByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return true, nil // Needs obtaining
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

// GetCertificateFunc returns a function suitable for tls.Config.GetCertificate.
func (p *ACMEProvider) GetCertificateFunc() func(*cryptotls.ClientHelloInfo) (*cryptotls.Certificate, error) {
	return p.GetCertificateWithLogging
}

// parseRetryAfter extracts the retry-after time from a rate limit error message.
func (p *ACMEProvider) parseRetryAfter(errStr string) time.Time {
	// Pattern: "retry after YYYY-MM-DD HH:MM:SS UTC"
	re := regexp.MustCompile(`retry after (\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}) UTC`)
	matches := re.FindStringSubmatch(errStr)
	if len(matches) >= 2 {
		t, err := time.Parse("2006-01-02 15:04:05", matches[1])
		if err == nil {
			return t.UTC()
		}
	}

	// Default: 1 hour
	return time.Now().Add(1 * time.Hour)
}

// Ensure interface compliance.
var _ ports.TLSProvider = (*ACMEProvider)(nil)

// ClientHelloInfo is a type alias for compatibility.
type ClientHelloInfo = cryptotls.ClientHelloInfo
