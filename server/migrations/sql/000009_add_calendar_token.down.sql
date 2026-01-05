-- Remove calendar_token from users table
DROP INDEX IF EXISTS idx_users_calendar_token;
ALTER TABLE users DROP COLUMN IF EXISTS calendar_token;
