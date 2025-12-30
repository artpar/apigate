-- Add updated_at column to api_keys table for consistency with other modules
ALTER TABLE api_keys ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;
