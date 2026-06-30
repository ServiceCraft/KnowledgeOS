-- 020_user_membership_version.sql: version stamp for membership cache invalidation

ALTER TABLE users ADD COLUMN IF NOT EXISTS membership_version INT NOT NULL DEFAULT 0;
