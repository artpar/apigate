-- Migration: Add auth_required column to routes table
-- This enables public (unauthenticated) routes for scenarios like reverse proxy to deployed apps

-- Add auth_required column with default TRUE (backward compatible - all existing routes require auth)
ALTER TABLE routes ADD COLUMN auth_required INTEGER NOT NULL DEFAULT 1;

-- Create index for efficient filtering of public routes
CREATE INDEX IF NOT EXISTS idx_routes_auth_required ON routes(auth_required);
