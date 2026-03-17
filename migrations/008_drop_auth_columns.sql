-- Migration 008: Remove auth-specific columns delegated to Zitadel/platform
-- Auth (password, OAuth IDs, phone) → Zitadel
-- Billing/referral (aff_code, aff_count, aff_quota, aff_history_quota, inviter_id) → lurus-platform

-- Make password nullable first to avoid INSERT failures during rolling deployments
ALTER TABLE users ALTER COLUMN password DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password SET DEFAULT '';

-- Drop auth and referral columns
ALTER TABLE users DROP COLUMN IF EXISTS password;
ALTER TABLE users DROP COLUMN IF EXISTS github_id;
ALTER TABLE users DROP COLUMN IF EXISTS discord_id;
ALTER TABLE users DROP COLUMN IF EXISTS oidc_id;
ALTER TABLE users DROP COLUMN IF EXISTS wechat_id;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_id;
ALTER TABLE users DROP COLUMN IF EXISTS linux_do_id;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
ALTER TABLE users DROP COLUMN IF EXISTS phone_verified;
ALTER TABLE users DROP COLUMN IF EXISTS aff_code;
ALTER TABLE users DROP COLUMN IF EXISTS aff_count;
ALTER TABLE users DROP COLUMN IF EXISTS aff_quota;
ALTER TABLE users DROP COLUMN IF EXISTS aff_history_quota;
ALTER TABLE users DROP COLUMN IF EXISTS inviter_id;

-- Drop deprecated tables (functionality moved to Zitadel / lurus-platform)
DROP TABLE IF EXISTS checkins;
DROP TABLE IF EXISTS twofas;
DROP TABLE IF EXISTS two_fa_backup_codes;
DROP TABLE IF EXISTS passkey_credentials;
DROP TABLE IF EXISTS passkey_sessions;
DROP TABLE IF EXISTS invitation_codes;
