CREATE TYPE task_status AS ENUM ('To Do', 'In Progress', 'Done');
CREATE TYPE task_sync_status AS ENUM ('Synced', 'Pending', 'Failed');
CREATE TYPE context_role AS ENUM ('me', 'other', 'system');
CREATE TYPE task_event_type AS ENUM ('Create', 'Assign', 'Status', 'Due', 'Delete', 'Restore', 'Comment');

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notion_page_id TEXT,
    title TEXT NOT NULL,
    status task_status NOT NULL DEFAULT 'To Do',
    sync_status task_sync_status NOT NULL DEFAULT 'Pending',
    group_id UUID,
    database_id TEXT,
    topic TEXT,
    due_at TIMESTAMPTZ,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    chat_jump_url TEXT,
    notion_url TEXT,
    archived BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_tasks_group_id ON tasks(group_id);
CREATE INDEX idx_tasks_creator_id ON tasks(creator_id);
CREATE INDEX idx_tasks_status ON tasks(status);

CREATE TABLE task_assignees (
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (task_id, user_id)
);

CREATE TABLE task_context_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    role context_role NOT NULL,
    author TEXT,
    text TEXT,
    tg_message_id BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_context_task_id ON task_context_snapshots(task_id);

CREATE TABLE task_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event task_event_type NOT NULL,
    before JSONB,
    after JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_task_id ON task_events(task_id);
