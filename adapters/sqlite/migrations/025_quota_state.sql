-- Quota state table for persistent quota tracking
-- Stores current period usage for fast synchronous quota enforcement
-- Without this, quota resets on server restart (users get "free" requests back)
CREATE TABLE IF NOT EXISTS quota_state (
    user_id TEXT NOT NULL,
    period_start DATETIME NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    compute_units REAL NOT NULL DEFAULT 0,
    bytes_used INTEGER NOT NULL DEFAULT 0,
    last_updated DATETIME NOT NULL,
    PRIMARY KEY (user_id, period_start)
);

-- Index for cleanup of old periods
CREATE INDEX IF NOT EXISTS idx_quota_state_period ON quota_state(period_start);
