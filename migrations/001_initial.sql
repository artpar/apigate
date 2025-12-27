-- APIGate Initial Schema
-- Supports both SQLite and PostgreSQL

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    stripe_customer_id TEXT,
    plan_id TEXT NOT NULL DEFAULT 'free',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_stripe_id ON users(stripe_customer_id);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT 'Default',
    scopes TEXT,  -- JSON array
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);

-- Usage records (aggregated per billing period)
CREATE TABLE IF NOT EXISTS usage_records (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id TEXT REFERENCES api_keys(id) ON DELETE SET NULL,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    compute_units REAL NOT NULL DEFAULT 0,
    bytes_in INTEGER NOT NULL DEFAULT 0,
    bytes_out INTEGER NOT NULL DEFAULT 0,
    UNIQUE(user_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_user_period ON usage_records(user_id, period_start);

-- Request logs (detailed, for analytics)
CREATE TABLE IF NOT EXISTS request_logs (
    id TEXT PRIMARY KEY,
    api_key_id TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    request_size INTEGER NOT NULL DEFAULT 0,
    response_size INTEGER NOT NULL DEFAULT 0,
    cost_multiplier REAL NOT NULL DEFAULT 1.0,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_request_logs_key ON request_logs(api_key_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_created ON request_logs(created_at);

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id TEXT NOT NULL,
    stripe_subscription_id TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    current_period_start TIMESTAMP NOT NULL,
    current_period_end TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cancelled_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_stripe ON subscriptions(stripe_subscription_id);

-- Invoices
CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stripe_invoice_id TEXT,
    period_start TIMESTAMP NOT NULL,
    period_end TIMESTAMP NOT NULL,
    subtotal INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'draft',
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_invoices_user ON invoices(user_id);

-- Rate limit tracking (for sliding window)
CREATE TABLE IF NOT EXISTS rate_limit_hits (
    api_key_id TEXT NOT NULL,
    window_start TIMESTAMP NOT NULL,
    hit_count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (api_key_id, window_start)
);

-- Cleanup old rate limit entries (run periodically)
-- DELETE FROM rate_limit_hits WHERE window_start < datetime('now', '-1 hour');
