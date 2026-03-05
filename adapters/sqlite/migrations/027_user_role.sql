-- Add role column to users table
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

-- Mark existing users created via setup as admin (ID starts with "admin_")
UPDATE users SET role = 'admin' WHERE id LIKE 'admin_%';
