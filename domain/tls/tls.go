// Package tls provides TLS certificate value types and pure validation functions.
// This package has NO dependencies on I/O or external packages.
package tls

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

// Mode represents the TLS configuration mode.
type Mode string

const (
	ModeNone   Mode = "none"   // No TLS (HTTP only)
	ModeACME   Mode = "acme"   // Automatic via ACME (Let's Encrypt)
	ModeManual Mode = "manual" // Manual certificate files
)

// IsValid returns true if the mode is known.
func (m Mode) IsValid() bool {
	switch m {
	case ModeNone, ModeACME, ModeManual:
		return true
	}
	return false
}

// Status represents the certificate status.
type Status string

const (
	StatusActive  Status = "active"
	StatusExpired Status = "expired"
	StatusRevoked Status = "revoked"
)

// IsValid returns true if the status is known.
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusExpired, StatusRevoked:
		return true
	}
	return false
}

// Certificate represents a stored TLS certificate (immutable value type).
type Certificate struct {
	ID             string
	Domain         string
	CertPEM        []byte
	ChainPEM       []byte
	KeyPEM         []byte
	IssuedAt       time.Time
	ExpiresAt      time.Time
	Issuer         string
	SerialNumber   string
	ACMEAccountURL string
	Status         Status
	RevokedAt      *time.Time
	RevokeReason   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// GenerateCertificateID creates a new certificate ID.
func GenerateCertificateID() string {
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	return "cert_" + hex.EncodeToString(idBytes)
}

// WithID returns a copy of the certificate with the ID set.
func (c Certificate) WithID(id string) Certificate {
	c.ID = id
	return c
}

// WithStatus returns a copy of the certificate with the Status set.
func (c Certificate) WithStatus(status Status) Certificate {
	c.Status = status
	return c
}

// IsExpired returns true if the certificate has expired.
func (c Certificate) IsExpired() bool {
	return time.Now().UTC().After(c.ExpiresAt)
}

// IsActive returns true if the certificate is active (not expired or revoked).
func (c Certificate) IsActive() bool {
	return c.Status == StatusActive && !c.IsExpired()
}

// DaysUntilExpiry returns the number of days until the certificate expires.
func (c Certificate) DaysUntilExpiry() int {
	remaining := c.ExpiresAt.Sub(time.Now().UTC())
	return int(remaining.Hours() / 24)
}

// NeedsRenewal returns true if the certificate should be renewed.
func (c Certificate) NeedsRenewal(renewalDays int) bool {
	return c.DaysUntilExpiry() <= renewalDays
}

// FullChain returns the full certificate chain (cert + chain).
func (c Certificate) FullChain() []byte {
	if len(c.ChainPEM) == 0 {
		return c.CertPEM
	}
	// Combine cert and chain
	return append(append(c.CertPEM, '\n'), c.ChainPEM...)
}

// Config represents TLS configuration (value type).
type Config struct {
	Enabled      bool
	Mode         Mode
	Domain       string
	Email        string
	CertPath     string
	KeyPath      string
	HTTPRedirect bool
	MinVersion   string
	ACMEStaging  bool
}

// Validate validates the TLS configuration (pure function).
type ConfigValidation struct {
	Valid  bool
	Errors map[string]string
}

// ValidateConfig validates TLS configuration (pure function).
func ValidateConfig(cfg Config) ConfigValidation {
	errors := make(map[string]string)

	if !cfg.Enabled {
		return ConfigValidation{Valid: true, Errors: errors}
	}

	if !cfg.Mode.IsValid() {
		errors["mode"] = "Invalid TLS mode"
	}

	switch cfg.Mode {
	case ModeACME:
		if cfg.Domain == "" {
			errors["domain"] = "Domain is required for ACME mode"
		} else if !isValidDomain(cfg.Domain) {
			errors["domain"] = "Invalid domain format"
		}
		if cfg.Email == "" {
			errors["email"] = "Email is required for ACME mode"
		} else if !isValidEmail(cfg.Email) {
			errors["email"] = "Invalid email format"
		}

	case ModeManual:
		if cfg.CertPath == "" {
			errors["cert_path"] = "Certificate path is required for manual mode"
		}
		if cfg.KeyPath == "" {
			errors["key_path"] = "Key path is required for manual mode"
		}
	}

	if cfg.MinVersion != "" && cfg.MinVersion != "1.2" && cfg.MinVersion != "1.3" {
		errors["min_version"] = "Min version must be 1.2 or 1.3"
	}

	return ConfigValidation{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ACMEChallenge represents an ACME HTTP-01 challenge (value type).
type ACMEChallenge struct {
	Token    string
	KeyAuth  string
	Domain   string
	Expires  time.Time
}

// IsExpired returns true if the challenge has expired.
func (c ACMEChallenge) IsExpired() bool {
	return time.Now().UTC().After(c.Expires)
}

// Path returns the challenge path for HTTP-01 challenge.
func (c ACMEChallenge) Path() string {
	return "/.well-known/acme-challenge/" + c.Token
}

// Helper functions (pure)

var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidDomain(domain string) bool {
	domain = strings.TrimSpace(domain)
	return domainRegex.MatchString(domain)
}

func isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	return emailRegex.MatchString(email)
}

// MinVersionToUint16 converts a string min version to TLS constant value.
// Returns 0 if invalid (caller should use default).
func MinVersionToUint16(minVersion string) uint16 {
	switch minVersion {
	case "1.2":
		return 0x0303 // tls.VersionTLS12
	case "1.3":
		return 0x0304 // tls.VersionTLS13
	default:
		return 0
	}
}
