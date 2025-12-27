-- Initial schema for APIGate

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    stripe_id TEXT,
    plan_id TEXT NOT NULL DEFAULT 'free',
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_stripe_id ON users(stripe_id);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- API Keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    hash BLOB NOT NULL,
    prefix TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    scopes TEXT, -- JSON array of allowed scopes
    expires_at DATETIME,
    revoked_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used DATETIME
);

CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);

-- Rate limit state table (ephemeral - can be cleared)
CREATE TABLE IF NOT EXISTS rate_limit_state (
    key_id TEXT PRIMARY KEY,
    count INTEGER NOT NULL DEFAULT 0,
    window_end DATETIME NOT NULL,
    burst_used INTEGER NOT NULL DEFAULT 0
);

-- Usage events table
CREATE TABLE IF NOT EXISTS usage_events (
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

CREATE INDEX IF NOT EXISTS idx_usage_events_user_id ON usage_events(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_key_id ON usage_events(key_id);
CREATE INDEX IF NOT EXISTS idx_usage_events_timestamp ON usage_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_events_user_timestamp ON usage_events(user_id, timestamp);

-- Usage summaries table (pre-aggregated for performance)
CREATE TABLE IF NOT EXISTS usage_summaries (
    user_id TEXT NOT NULL,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    compute_units REAL NOT NULL DEFAULT 0,
    bytes_in INTEGER NOT NULL DEFAULT 0,
    bytes_out INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    avg_latency_ms INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, period_start)
);

CREATE INDEX IF NOT EXISTS idx_usage_summaries_user_period ON usage_summaries(user_id, period_start, period_end);

-- Subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL, -- 'stripe', 'paddle', 'lemonsqueezy'
    external_id TEXT NOT NULL,
    plan_id TEXT NOT NULL,
    status TEXT NOT NULL,
    current_period_start DATETIME NOT NULL,
    current_period_end DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_external_id ON subscriptions(external_id);

-- Invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    subscription_id TEXT REFERENCES subscriptions(id),
    provider TEXT NOT NULL,
    external_id TEXT,
    amount INTEGER NOT NULL, -- cents
    currency TEXT NOT NULL DEFAULT 'usd',
    status TEXT NOT NULL,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    paid_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices(user_id);
CREATE INDEX IF NOT EXISTS idx_invoices_external_id ON invoices(external_id);
