-- Add password_hash column to users table for web UI authentication
ALTER TABLE users ADD COLUMN password_hash BLOB;
