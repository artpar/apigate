-- Add quota_bypass column for service account keys
-- Service accounts bypass quota limits for admin/infrastructure operations

ALTER TABLE api_keys ADD COLUMN quota_bypass BOOLEAN NOT NULL DEFAULT FALSE;

-- Index for efficient service account queries
CREATE INDEX IF NOT EXISTS idx_api_keys_quota_bypass ON api_keys(quota_bypass) WHERE quota_bypass = TRUE;
