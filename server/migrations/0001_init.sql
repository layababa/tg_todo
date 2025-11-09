-- 初始化任务、指派、用户及消息表结构
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    username TEXT,
    display_name TEXT NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tasks (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    creator_id BIGINT NOT NULL REFERENCES users(id),
    source_message_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS task_assignees (
    task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id),
    PRIMARY KEY (task_id, user_id)
);

CREATE TABLE IF NOT EXISTS telegram_messages (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE,
    chat_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed 示例用户
INSERT INTO users (id, username, display_name)
VALUES
    (1000, 'pm_lead', '产品经理'),
    (2000, 'fe_lead', '前端负责人'),
    (2001, 'be_lead', '后端负责人'),
    (3000, 'designer', '设计师')
ON CONFLICT (id) DO NOTHING;

-- Seed 示例任务
INSERT INTO tasks (id, title, description, status, creator_id, source_message_url)
VALUES
    (1, '审核群组内被 @ 的待办项', '引用 + @bot + @成员 生成的任务示例', 'pending', 1000, 'https://t.me/c/12345/67890'),
    (2, '为 Mini App 列表页增加 uiverse.io 完成动画', NULL, 'completed', 3000, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO task_assignees (task_id, user_id)
VALUES
    (1, 2000),
    (1, 2001),
    (2, 2000)
ON CONFLICT (task_id, user_id) DO NOTHING;
