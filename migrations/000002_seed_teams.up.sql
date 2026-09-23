INSERT INTO teams (id, name, created_at, updated_at) VALUES
('e4b4f5aa-8fb8-4e33-911f-c0d12e879a51', 'Engineering', NOW(), NOW()),
('7b2354c0-7b26-4b68-98e6-1200fa445ef6', 'Product', NOW(), NOW()),
('84920959-1959-4d37-8898-8e6d0bb1a723', 'Quality Assurance', NOW(), NOW()),
('9f1d0443-3b1a-4710-85f8-842cd4c7d0d0', 'Design', NOW(), NOW()),
('c116c498-39d8-4f70-a35b-c2e74e622ef7', 'DevOps', NOW(), NOW())
ON CONFLICT (name) DO UPDATE SET
    id = EXCLUDED.id,
    updated_at = NOW();
