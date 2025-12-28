-- Routes and Upstreams schema for API Gateway functionality

-- Upstreams table - backend service configurations
CREATE TABLE IF NOT EXISTS upstreams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Target
    base_url TEXT NOT NULL,
    timeout_ms INTEGER NOT NULL DEFAULT 30000,

    -- Connection pooling
    max_idle_conns INTEGER NOT NULL DEFAULT 100,
    idle_conn_timeout_ms INTEGER NOT NULL DEFAULT 90000,

    -- Authentication injection
    auth_type TEXT NOT NULL DEFAULT 'none', -- none, header, bearer, basic
    auth_header TEXT,                        -- Header name for auth_type=header
    auth_value_encrypted BLOB,               -- Encrypted auth value

    -- Metadata
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_upstreams_enabled ON upstreams(enabled);
CREATE INDEX IF NOT EXISTS idx_upstreams_name ON upstreams(name);

-- Routes table - request routing rules
CREATE TABLE IF NOT EXISTS routes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',

    -- Matching criteria
    path_pattern TEXT NOT NULL,              -- /api/v1/*, /users/{id}, regex pattern
    match_type TEXT NOT NULL DEFAULT 'prefix', -- exact, prefix, regex
    methods TEXT,                            -- JSON array: ["GET", "POST"], null = all
    headers TEXT,                            -- JSON array of HeaderMatch objects

    -- Target configuration
    upstream_id TEXT NOT NULL REFERENCES upstreams(id) ON DELETE RESTRICT,
    path_rewrite TEXT,                       -- Expr expression for path rewriting
    method_override TEXT,                    -- Override request method

    -- Transformations (JSON)
    request_transform TEXT,                  -- JSON Transform object
    response_transform TEXT,                 -- JSON Transform object

    -- Metering
    metering_expr TEXT NOT NULL DEFAULT '1', -- Expr to extract usage value
    metering_mode TEXT NOT NULL DEFAULT 'request', -- request, response_field, bytes, custom

    -- Protocol behavior
    protocol TEXT NOT NULL DEFAULT 'http',   -- http, http_stream, sse, websocket

    -- Metadata
    priority INTEGER NOT NULL DEFAULT 0,     -- Higher = evaluated first
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_routes_enabled_priority ON routes(enabled, priority DESC);
CREATE INDEX IF NOT EXISTS idx_routes_path_pattern ON routes(path_pattern);
CREATE INDEX IF NOT EXISTS idx_routes_upstream_id ON routes(upstream_id);
CREATE INDEX IF NOT EXISTS idx_routes_name ON routes(name);

-- Default upstream (optional - for backward compatibility)
-- Insert a default "passthrough" upstream if migrating existing installation
-- This can be customized per deployment
