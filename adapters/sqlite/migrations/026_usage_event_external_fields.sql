-- Add external event fields to usage_events table.
-- These fields are populated by events submitted via POST /api/v1/meter.

ALTER TABLE usage_events ADD COLUMN event_type TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_events ADD COLUMN resource_id TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_events ADD COLUMN resource_type TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_events ADD COLUMN quantity REAL NOT NULL DEFAULT 1.0;
ALTER TABLE usage_events ADD COLUMN source TEXT NOT NULL DEFAULT 'proxy';
ALTER TABLE usage_events ADD COLUMN source_name TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_events ADD COLUMN metadata TEXT; -- JSON object, nullable

CREATE INDEX IF NOT EXISTS idx_usage_events_event_type ON usage_events(event_type);
CREATE INDEX IF NOT EXISTS idx_usage_events_source ON usage_events(source);
