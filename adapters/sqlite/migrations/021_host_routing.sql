-- Add host-based routing columns to routes table
-- Enables multi-tenant and subdomain-based API routing

ALTER TABLE routes ADD COLUMN host_pattern TEXT NOT NULL DEFAULT '';
ALTER TABLE routes ADD COLUMN host_match_type TEXT NOT NULL DEFAULT '';

-- Index for host pattern lookups (only for routes with host patterns)
CREATE INDEX IF NOT EXISTS idx_routes_host_pattern ON routes(host_pattern) WHERE host_pattern != '';
