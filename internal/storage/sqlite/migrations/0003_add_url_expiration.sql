-- 0003_add_url_expiration
-- ALTER TABLE has no IF NOT EXISTS in SQLite; this is safe because the runner
-- records applied migrations atomically with their application.
ALTER TABLE url ADD COLUMN expires_at TIMESTAMP;
