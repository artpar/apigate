-- Migration 010: Add trial_days column to plans table
-- Supports trial period feature for subscription plans

ALTER TABLE plans ADD COLUMN trial_days INTEGER NOT NULL DEFAULT 0;
