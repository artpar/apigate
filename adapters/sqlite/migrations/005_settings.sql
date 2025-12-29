-- Settings table for storing all application configuration
-- This replaces file-based config and enables runtime configuration

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    encrypted INTEGER DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for common access patterns
CREATE INDEX IF NOT EXISTS idx_settings_prefix ON settings(key);

-- Insert default settings if not exist
INSERT OR IGNORE INTO settings (key, value, encrypted) VALUES
    ('server.host', '0.0.0.0', 0),
    ('server.port', '8080', 0),
    ('server.read_timeout', '30s', 0),
    ('server.write_timeout', '60s', 0),
    ('portal.enabled', 'false', 0),
    ('portal.app_name', 'APIGate', 0),
    ('email.provider', 'none', 0),
    ('payment.provider', 'none', 0),
    ('auth.key_prefix', 'ak_', 0),
    ('auth.session_ttl', '168h', 0),
    ('ratelimit.enabled', 'true', 0),
    ('ratelimit.burst_tokens', '5', 0),
    ('ratelimit.window_secs', '60', 0),
    ('upstream.timeout', '30s', 0),
    ('upstream.max_idle_conns', '100', 0),
    ('upstream.idle_conn_timeout', '90s', 0);

-- Plans table (moved from config to database)
CREATE TABLE IF NOT EXISTS plans (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    rate_limit_per_minute INTEGER DEFAULT 60,
    requests_per_month INTEGER DEFAULT 1000,
    price_monthly INTEGER DEFAULT 0,
    overage_price INTEGER DEFAULT 0,
    features TEXT, -- JSON array of feature strings
    stripe_price_id TEXT,
    paddle_price_id TEXT,
    lemon_variant_id TEXT,
    is_default INTEGER DEFAULT 0,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default free plan
INSERT OR IGNORE INTO plans (id, name, rate_limit_per_minute, requests_per_month, is_default, enabled) VALUES
    ('free', 'Free', 60, 1000, 1, 1);
