-- Add proper foreign key constraints with ON DELETE actions
-- SQLite requires table recreation to add constraints to existing tables

-- 1. Recreate auth_tokens with FK constraint
CREATE TABLE IF NOT EXISTS auth_tokens_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    token_type TEXT NOT NULL,
    token_hash BLOB NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO auth_tokens_new
    SELECT id, user_id, email, token_type, token_hash, expires_at, used_at, created_at
    FROM auth_tokens WHERE user_id IN (SELECT id FROM users);

DROP TABLE IF EXISTS auth_tokens;
ALTER TABLE auth_tokens_new RENAME TO auth_tokens;

CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_type ON auth_tokens(user_id, token_type);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_expires ON auth_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_hash ON auth_tokens(token_hash);

-- 2. Recreate user_sessions with FK constraint
CREATE TABLE IF NOT EXISTS user_sessions_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO user_sessions_new
    SELECT id, user_id, email, ip_address, user_agent, expires_at, created_at
    FROM user_sessions WHERE user_id IN (SELECT id FROM users);

DROP TABLE IF EXISTS user_sessions;
ALTER TABLE user_sessions_new RENAME TO user_sessions;

CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires ON user_sessions(expires_at);

-- 3. Recreate rate_limit_state with FK constraint to api_keys
CREATE TABLE IF NOT EXISTS rate_limit_state_new (
    key_id TEXT PRIMARY KEY REFERENCES api_keys(id) ON DELETE CASCADE,
    count INTEGER NOT NULL DEFAULT 0,
    window_end DATETIME NOT NULL,
    burst_used INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO rate_limit_state_new
    SELECT key_id, count, window_end, burst_used
    FROM rate_limit_state WHERE key_id IN (SELECT id FROM api_keys);

DROP TABLE IF EXISTS rate_limit_state;
ALTER TABLE rate_limit_state_new RENAME TO rate_limit_state;

-- 4. Add ON DELETE CASCADE to api_keys (recreate)
CREATE TABLE IF NOT EXISTS api_keys_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hash BLOB NOT NULL,
    prefix TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    scopes TEXT,
    expires_at DATETIME,
    revoked_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO api_keys_new
    SELECT id, user_id, hash, prefix, name, scopes, expires_at, revoked_at, created_at, last_used,
           COALESCE(updated_at, created_at)
    FROM api_keys WHERE user_id IN (SELECT id FROM users);

DROP TABLE IF EXISTS api_keys;
ALTER TABLE api_keys_new RENAME TO api_keys;

CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);

-- 5. Add ON DELETE CASCADE to subscriptions (recreate)
CREATE TABLE IF NOT EXISTS subscriptions_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    external_id TEXT NOT NULL,
    plan_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_period_start DATETIME NOT NULL,
    current_period_end DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO subscriptions_new
    SELECT id, user_id, provider, external_id, plan_id, status,
           current_period_start, current_period_end, created_at, updated_at
    FROM subscriptions WHERE user_id IN (SELECT id FROM users);

DROP TABLE IF EXISTS subscriptions;
ALTER TABLE subscriptions_new RENAME TO subscriptions;

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_external_id ON subscriptions(external_id);

-- 6. Add ON DELETE CASCADE to invoices (recreate)
CREATE TABLE IF NOT EXISTS invoices_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id TEXT REFERENCES subscriptions(id) ON DELETE SET NULL,
    provider TEXT NOT NULL,
    external_id TEXT,
    amount INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'usd',
    status TEXT NOT NULL,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    paid_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO invoices_new
    SELECT id, user_id, subscription_id, provider, external_id, amount, currency,
           status, period_start, period_end, paid_at, created_at
    FROM invoices WHERE user_id IN (SELECT id FROM users);

DROP TABLE IF EXISTS invoices;
ALTER TABLE invoices_new RENAME TO invoices;

CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices(user_id);
CREATE INDEX IF NOT EXISTS idx_invoices_external_id ON invoices(external_id);

-- 7. Add FK constraints to usage_events (recreate for consistency)
CREATE TABLE IF NOT EXISTS usage_events_new (
    id TEXT PRIMARY KEY,
    key_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    request_bytes INTEGER NOT NULL DEFAULT 0,
    response_bytes INTEGER NOT NULL DEFAULT 0,
    cost_multiplier REAL NOT NULL DEFAULT 1.0,
    ip_address TEXT,
    user_agent TEXT,
    timestamp DATETIME NOT NULL
);

INSERT OR IGNORE INTO usage_events_new
    SELECT * FROM usage_events;

DROP TABLE IF EXISTS usage_events;
ALTER TABLE usage_events_new RENAME TO usage_events;

CREATE INDEX IF NOT EXISTS idx_usage_events_user_id ON usage_events(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_key_id ON usage_events(key_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_timestamp ON usage_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_events_user_timestamp ON usage_events(user_id, timestamp);
