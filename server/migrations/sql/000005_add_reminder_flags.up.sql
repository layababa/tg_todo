-- +migrate Up
ALTER TABLE tasks ADD COLUMN reminder_1h_sent BOOLEAN DEFAULT FALSE;
ALTER TABLE tasks ADD COLUMN reminder_due_sent BOOLEAN DEFAULT FALSE;

-- +migrate Down
ALTER TABLE tasks DROP COLUMN reminder_1h_sent;
ALTER TABLE tasks DROP COLUMN reminder_due_sent;

