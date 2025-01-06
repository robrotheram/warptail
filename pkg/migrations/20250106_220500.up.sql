ALTER TABLE users ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT 'internal';
--bun:split
UPDATE users SET type = 'internal';