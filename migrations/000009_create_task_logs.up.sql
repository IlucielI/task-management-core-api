CREATE TABLE IF NOT EXISTS task_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL REFERENCES task_actions(code) ON DELETE RESTRICT,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    from_assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    to_assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    from_status VARCHAR(50) NULL REFERENCES task_statuses(code) ON DELETE SET NULL,
    to_status VARCHAR(50) NULL REFERENCES task_statuses(code) ON DELETE SET NULL,
    notes TEXT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_logs_task_created ON task_logs (task_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_logs_actor_id ON task_logs (actor_id);
