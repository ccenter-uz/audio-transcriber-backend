ALTER TABLE users ADD COLUMN can_login BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users SET can_login = TRUE WHERE role = 'admin' AND deleted_at = 0;
