-- Migration 007: Add quota enforcement fields to plans
-- These fields control how quota limits are enforced per plan

-- Add quota enforcement mode: 'hard' (reject), 'warn' (headers only), 'soft' (bill overage)
ALTER TABLE plans ADD COLUMN quota_enforce_mode TEXT DEFAULT 'hard';

-- Add grace percentage: allows slight overage before hard block (e.g., 0.05 = 5%)
ALTER TABLE plans ADD COLUMN quota_grace_pct REAL DEFAULT 0.05;
