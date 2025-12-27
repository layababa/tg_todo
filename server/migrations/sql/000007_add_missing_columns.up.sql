-- Add description to tasks
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS description TEXT;

-- Add default_database_id to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS default_database_id TEXT;
