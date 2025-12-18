CREATE TABLE IF NOT EXISTS telegram_updates (
    id BIGSERIAL PRIMARY KEY,
    update_id BIGINT NOT NULL UNIQUE,
    raw_data JSONB NOT NULL,
    processed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_telegram_updates_update_id ON telegram_updates(update_id);
CREATE INDEX idx_telegram_updates_created_at ON telegram_updates(created_at);
