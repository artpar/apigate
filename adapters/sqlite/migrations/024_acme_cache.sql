-- ACME cache table for storing account keys and other autocert cache data
-- This ensures ACME account keys survive restarts and prevent Let's Encrypt rate limiting
CREATE TABLE IF NOT EXISTS acme_cache (
    key TEXT PRIMARY KEY,
    data BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster lookups by key prefix (e.g., finding all account keys)
CREATE INDEX IF NOT EXISTS idx_acme_cache_key ON acme_cache(key);
