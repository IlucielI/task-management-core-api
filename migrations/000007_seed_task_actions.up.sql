INSERT INTO task_actions (code, name, description, created_at) VALUES
('ASSIGN', 'Task Assigned', 'Task assignment to another team member', NOW()),
('CREATE', 'Task Created', 'Initial task creation', NOW()),
('UPDATE', 'Task Updated', 'Task title or description updated', NOW()),
('STATUS_UPDATE', 'Task Status Updated', 'Task workflow status transition', NOW()),
('DELETE', 'Task Deleted', 'Task removal', NOW())
ON CONFLICT (code) DO NOTHING;
