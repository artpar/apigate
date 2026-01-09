-- Migration 016: Admin invites for web-based admin onboarding
-- Allows existing admins to invite new admins via email link

CREATE TABLE IF NOT EXISTS admin_invites (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    token_hash BLOB NOT NULL UNIQUE,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    used_at DATETIME,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_admin_invites_email ON admin_invites(email);
CREATE INDEX IF NOT EXISTS idx_admin_invites_expires_at ON admin_invites(expires_at);
