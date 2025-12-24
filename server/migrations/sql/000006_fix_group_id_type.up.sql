-- Fix group_id type mismatch
-- groups.id is TEXT (Telegram Chat ID), but tasks.group_id was UUID
ALTER TABLE tasks ALTER COLUMN group_id TYPE TEXT;
