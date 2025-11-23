-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tg_id BIGINT NOT NULL UNIQUE,
    tg_username TEXT,
    name TEXT NOT NULL,
    photo_url TEXT,
    avatar TEXT, -- Deprecated, use photo_url
    timezone TEXT NOT NULL DEFAULT 'UTC+0',
    notion_connected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create index on tg_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_users_tg_id ON users(tg_id) WHERE deleted_at IS NULL;

-- Create user_notion_tokens table
CREATE TABLE IF NOT EXISTS user_notion_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_token_enc TEXT NOT NULL,
    refresh_token_enc TEXT,
    expires_at TIMESTAMPTZ,
    workspace_id TEXT NOT NULL,
    workspace_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Create index on user_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_user_notion_tokens_user_id ON user_notion_tokens(user_id) WHERE deleted_at IS NULL;

-- Create a function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to auto-update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_notion_tokens_updated_at BEFORE UPDATE ON user_notion_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
