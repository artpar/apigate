-- Add metering_unit column to routes table
-- This allows each route to specify its own display unit (requests, tokens, data_points, bytes)
-- instead of using a global setting

ALTER TABLE routes ADD COLUMN metering_unit TEXT NOT NULL DEFAULT 'requests';
