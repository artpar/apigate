-- Entitlements table - defines feature flags and capabilities
CREATE TABLE IF NOT EXISTS entitlements (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    display_name TEXT,
    description TEXT,
    category TEXT DEFAULT 'feature',
    value_type TEXT DEFAULT 'boolean',
    default_value TEXT DEFAULT 'true',
    header_name TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_entitlements_name ON entitlements(name);
CREATE INDEX IF NOT EXISTS idx_entitlements_category ON entitlements(category);
CREATE INDEX IF NOT EXISTS idx_entitlements_enabled ON entitlements(enabled);

-- Plan-Entitlement mappings - many-to-many relationship
CREATE TABLE IF NOT EXISTS plan_entitlements (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    entitlement_id TEXT NOT NULL,
    value TEXT,
    notes TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(plan_id, entitlement_id)
);

CREATE INDEX IF NOT EXISTS idx_plan_entitlements_plan ON plan_entitlements(plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_entitlements_entitlement ON plan_entitlements(entitlement_id);
CREATE INDEX IF NOT EXISTS idx_plan_entitlements_enabled ON plan_entitlements(enabled);

-- Insert default entitlements that map to common API gateway features
INSERT OR IGNORE INTO entitlements (id, name, display_name, description, category, value_type, default_value, header_name, enabled) VALUES
    ('ent-api-access', 'api.access', 'API Access', 'Basic API access', 'api', 'boolean', 'true', NULL, 1),
    ('ent-api-streaming', 'api.streaming', 'Streaming Responses', 'SSE and streaming API responses', 'api', 'boolean', 'true', 'X-Has-Streaming', 1),
    ('ent-api-batch', 'api.batch', 'Batch Requests', 'Batch API requests', 'api', 'boolean', 'true', 'X-Has-Batch', 1),
    ('ent-support-priority', 'support.priority', 'Priority Support', 'Priority support access', 'support', 'boolean', 'true', NULL, 1),
    ('ent-support-sla', 'support.sla', 'SLA Guarantee', 'Service level agreement coverage', 'support', 'boolean', 'true', NULL, 1);

-- Assign default entitlements to the free plan
INSERT OR IGNORE INTO plan_entitlements (id, plan_id, entitlement_id, value, enabled) VALUES
    ('pe-free-api-access', 'free', 'ent-api-access', 'true', 1);
