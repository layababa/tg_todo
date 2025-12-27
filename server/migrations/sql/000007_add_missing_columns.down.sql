-- Remove default_database_id from users
ALTER TABLE users DROP COLUMN IF NOT EXISTS default_database_id;

-- Remove description from tasks
ALTER TABLE tasks DROP COLUMN IF NOT EXISTS description;
