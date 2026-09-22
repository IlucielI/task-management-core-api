INSERT INTO task_statuses (code, name, description, created_at) VALUES
('todo', 'To Do', 'Task is created and pending work', NOW()),
('in_progress', 'In Progress', 'Task is currently being worked on', NOW()),
('code_review', 'Code Review', 'Task changes are in code review', NOW()),
('ready_for_qa', 'Ready for QA', 'Task is ready for quality assurance testing', NOW()),
('done', 'Done', 'Task is completed and verified', NOW())
ON CONFLICT (code) DO NOTHING;
