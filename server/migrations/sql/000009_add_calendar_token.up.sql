-- Add calendar_token to users table for calendar subscription
ALTER TABLE users ADD COLUMN calendar_token TEXT UNIQUE;

-- Create index for faster lookup by token
CREATE INDEX idx_users_calendar_token ON users(calendar_token) WHERE calendar_token IS NOT NULL;
