-- Add id column to settings table for module system compatibility
-- SQLite doesn't support adding PRIMARY KEY to existing tables, so we recreate

-- Create new table with id column
CREATE TABLE IF NOT EXISTS settings_new (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    encrypted INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Copy existing data, using key as id for existing rows
INSERT OR IGNORE INTO settings_new (id, key, value, encrypted, updated_at)
SELECT key, key, value, encrypted, updated_at FROM settings;

-- Drop old table
DROP TABLE IF EXISTS settings;

-- Rename new table
ALTER TABLE settings_new RENAME TO settings;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_settings_key ON settings(key);
