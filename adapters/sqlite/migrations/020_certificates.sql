-- TLS certificates table - database-backed for horizontal scaling
-- Stores ACME and manual certificates
CREATE TABLE IF NOT EXISTS certificates (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL,
    cert_pem BLOB NOT NULL,
    chain_pem BLOB,
    key_pem BLOB NOT NULL,
    issued_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    issuer TEXT,
    serial_number TEXT,
    acme_account_url TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    revoked_at DATETIME,
    revoke_reason TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index on domain for fast lookups
-- Note: domain column is not UNIQUE because we may have multiple certs
-- for the same domain (expired ones, different wildcard levels, etc.)
CREATE INDEX IF NOT EXISTS idx_certificates_domain ON certificates(domain);
CREATE INDEX IF NOT EXISTS idx_certificates_expires ON certificates(expires_at);
CREATE INDEX IF NOT EXISTS idx_certificates_status ON certificates(status);

-- Create unique index for active certificates per domain
-- This ensures only one active cert per domain
CREATE UNIQUE INDEX IF NOT EXISTS idx_certificates_domain_active
ON certificates(domain) WHERE status = 'active';
