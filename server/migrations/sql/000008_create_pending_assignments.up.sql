CREATE TABLE IF NOT EXISTS pending_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tg_username TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pending_assignments_username ON pending_assignments(tg_username);
CREATE INDEX idx_pending_assignments_task_id ON pending_assignments(task_id);
