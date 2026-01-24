package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/artpar/apigate/domain/tls"
	"github.com/artpar/apigate/ports"
)

// CertificateStore implements ports.CertificateStore using SQLite.
// Database-backed for horizontal scaling (stateless servers).
type CertificateStore struct {
	db *DB
}

// NewCertificateStore creates a new SQLite certificate store.
func NewCertificateStore(db *DB) *CertificateStore {
	return &CertificateStore{db: db}
}

// Get retrieves a certificate by ID.
func (s *CertificateStore) Get(ctx context.Context, id string) (tls.Certificate, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		       issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		       created_at, updated_at
		FROM certificates
		WHERE id = ?
	`, id)
	return scanCertificate(row)
}

// GetByDomain retrieves a certificate by domain.
// Supports exact match and wildcard lookup.
// For example, looking up "api.example.com" will also match "*.example.com".
func (s *CertificateStore) GetByDomain(ctx context.Context, domain string) (tls.Certificate, error) {
	// First try exact match
	row := s.db.QueryRowContext(ctx, `
		SELECT id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		       issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		       created_at, updated_at
		FROM certificates
		WHERE domain = ? AND status = 'active'
		ORDER BY expires_at DESC
		LIMIT 1
	`, domain)

	cert, err := scanCertificate(row)
	if err == nil {
		return cert, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return tls.Certificate{}, err
	}

	// Try wildcard match - extract parent domain and check for *.parent.domain
	wildcardDomain := extractWildcardDomain(domain)
	if wildcardDomain == "" {
		return tls.Certificate{}, ErrNotFound
	}

	row = s.db.QueryRowContext(ctx, `
		SELECT id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		       issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		       created_at, updated_at
		FROM certificates
		WHERE domain = ? AND status = 'active'
		ORDER BY expires_at DESC
		LIMIT 1
	`, wildcardDomain)

	return scanCertificate(row)
}

// extractWildcardDomain converts "sub.example.com" to "*.example.com".
func extractWildcardDomain(domain string) string {
	// Find first dot
	for i, c := range domain {
		if c == '.' {
			if i < len(domain)-1 {
				return "*" + domain[i:]
			}
			return ""
		}
	}
	return ""
}

// Create stores a new certificate.
func (s *CertificateStore) Create(ctx context.Context, cert tls.Certificate) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO certificates (id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		                          issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		                          created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, cert.ID, cert.Domain, cert.CertPEM, cert.ChainPEM, cert.KeyPEM,
		cert.IssuedAt, cert.ExpiresAt, cert.Issuer, cert.SerialNumber,
		nullStringCert(cert.ACMEAccountURL), cert.Status, cert.RevokedAt,
		nullStringCert(cert.RevokeReason), cert.CreatedAt, cert.UpdatedAt)
	return err
}

// Update modifies a certificate (e.g., renewal).
func (s *CertificateStore) Update(ctx context.Context, cert tls.Certificate) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE certificates
		SET cert_pem = ?, chain_pem = ?, key_pem = ?, issued_at = ?, expires_at = ?,
		    issuer = ?, serial_number = ?, acme_account_url = ?, status = ?,
		    revoked_at = ?, revoke_reason = ?, updated_at = ?
		WHERE id = ?
	`, cert.CertPEM, cert.ChainPEM, cert.KeyPEM, cert.IssuedAt, cert.ExpiresAt,
		cert.Issuer, cert.SerialNumber, nullStringCert(cert.ACMEAccountURL),
		cert.Status, cert.RevokedAt, nullStringCert(cert.RevokeReason),
		time.Now().UTC(), cert.ID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a certificate.
func (s *CertificateStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM certificates WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns all certificates.
func (s *CertificateStore) List(ctx context.Context) ([]tls.Certificate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		       issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		       created_at, updated_at
		FROM certificates
		ORDER BY domain ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCertificates(rows)
}

// ListExpiring returns certificates expiring within N days.
func (s *CertificateStore) ListExpiring(ctx context.Context, days int) ([]tls.Certificate, error) {
	expirationThreshold := time.Now().UTC().AddDate(0, 0, days)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		       issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		       created_at, updated_at
		FROM certificates
		WHERE status = 'active' AND expires_at <= ? AND expires_at > ?
		ORDER BY expires_at ASC
	`, expirationThreshold, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCertificates(rows)
}

// ListExpired returns expired certificates.
func (s *CertificateStore) ListExpired(ctx context.Context) ([]tls.Certificate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, domain, cert_pem, chain_pem, key_pem, issued_at, expires_at,
		       issuer, serial_number, acme_account_url, status, revoked_at, revoke_reason,
		       created_at, updated_at
		FROM certificates
		WHERE expires_at < ?
		ORDER BY expires_at DESC
	`, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanCertificates(rows)
}

func scanCertificate(row *sql.Row) (tls.Certificate, error) {
	var cert tls.Certificate
	var chainPEM, issuer, serialNumber, acmeAccountURL, revokeReason sql.NullString
	var revokedAt sql.NullTime

	err := row.Scan(
		&cert.ID, &cert.Domain, &cert.CertPEM, &chainPEM, &cert.KeyPEM,
		&cert.IssuedAt, &cert.ExpiresAt, &issuer, &serialNumber, &acmeAccountURL,
		&cert.Status, &revokedAt, &revokeReason, &cert.CreatedAt, &cert.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return tls.Certificate{}, ErrNotFound
	}
	if err != nil {
		return tls.Certificate{}, err
	}

	if chainPEM.Valid {
		cert.ChainPEM = []byte(chainPEM.String)
	}
	cert.Issuer = issuer.String
	cert.SerialNumber = serialNumber.String
	cert.ACMEAccountURL = acmeAccountURL.String
	cert.RevokeReason = revokeReason.String
	if revokedAt.Valid {
		cert.RevokedAt = &revokedAt.Time
	}

	return cert, nil
}

func scanCertificates(rows *sql.Rows) ([]tls.Certificate, error) {
	var certs []tls.Certificate
	for rows.Next() {
		var cert tls.Certificate
		var chainPEM, issuer, serialNumber, acmeAccountURL, revokeReason sql.NullString
		var revokedAt sql.NullTime

		err := rows.Scan(
			&cert.ID, &cert.Domain, &cert.CertPEM, &chainPEM, &cert.KeyPEM,
			&cert.IssuedAt, &cert.ExpiresAt, &issuer, &serialNumber, &acmeAccountURL,
			&cert.Status, &revokedAt, &revokeReason, &cert.CreatedAt, &cert.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if chainPEM.Valid {
			cert.ChainPEM = []byte(chainPEM.String)
		}
		cert.Issuer = issuer.String
		cert.SerialNumber = serialNumber.String
		cert.ACMEAccountURL = acmeAccountURL.String
		cert.RevokeReason = revokeReason.String
		if revokedAt.Valid {
			cert.RevokedAt = &revokedAt.Time
		}

		certs = append(certs, cert)
	}
	return certs, rows.Err()
}

func nullStringCert(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// -----------------------------------------------------------------------------
// ACMECacheStore Implementation
// -----------------------------------------------------------------------------

// GetCache retrieves cached ACME data by key.
func (s *CertificateStore) GetCache(ctx context.Context, key string) ([]byte, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT data FROM acme_cache WHERE key = ?
	`, key)

	var data []byte
	err := row.Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// PutCache stores ACME data with the given key.
func (s *CertificateStore) PutCache(ctx context.Context, key string, data []byte) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO acme_cache (key, data, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET
			data = excluded.data,
			updated_at = CURRENT_TIMESTAMP
	`, key, data)
	return err
}

// DeleteCache removes cached ACME data by key.
func (s *CertificateStore) DeleteCache(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM acme_cache WHERE key = ?`, key)
	return err
}

// Ensure interface compliance.
var _ ports.CertificateStore = (*CertificateStore)(nil)
var _ ports.ACMECacheStore = (*CertificateStore)(nil)
