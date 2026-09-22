DELETE FROM task_statuses WHERE code IN ('todo', 'in_progress', 'code_review', 'ready_for_qa', 'done');
