-- Rollback: restore group_id to UUID type
ALTER TABLE tasks ALTER COLUMN group_id TYPE UUID USING group_id::UUID;
