-- Add documentation fields to routes for customer-facing API docs
-- These fields are used to generate API reference, examples, and try-it pages

ALTER TABLE routes ADD COLUMN example_request TEXT NOT NULL DEFAULT '';
ALTER TABLE routes ADD COLUMN example_response TEXT NOT NULL DEFAULT '';

-- Note: description column already exists in routes table
