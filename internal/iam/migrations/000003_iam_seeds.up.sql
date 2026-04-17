-- Seed Roles
INSERT INTO iam.roles (id, code, name, description, is_system, created_at, updated_at) VALUES 
('01H00000000000000000000001', 'user', 'User', 'Standard baseline user', true, NOW(), NOW()),
('01H00000000000000000000002', 'admin', 'Admin', 'System administrator', true, NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Seed Permissions
INSERT INTO iam.permissions (id, code, description, created_at, updated_at) VALUES 
('01H00000000000000000001001', 'iam:user:read', 'Read user profiles', NOW(), NOW()),
('01H00000000000000000001002', 'iam:user:write', 'Modify user profiles', NOW(), NOW()),
('01H00000000000000000001003', 'iam:role:read', 'Read roles and hierarchies', NOW(), NOW()),
('01H00000000000000000001004', 'iam:role:assign', 'Assign roles to users', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Map basic User permissions
INSERT INTO iam.role_permissions (role_id, permission_id) VALUES 
('01H00000000000000000000001', '01H00000000000000000001001')
ON CONFLICT DO NOTHING;

-- Map Admin permissions
INSERT INTO iam.role_permissions (role_id, permission_id) VALUES 
('01H00000000000000000000002', '01H00000000000000000001001'),
('01H00000000000000000000002', '01H00000000000000000001002'),
('01H00000000000000000000002', '01H00000000000000000001003'),
('01H00000000000000000000002', '01H00000000000000000001004')
ON CONFLICT DO NOTHING;

-- Seed Root User (Password is 'rootpassword' mapped via Argon2id)
INSERT INTO iam.users (id, username, email, full_name, password_hash, status, created_at, updated_at) VALUES 
('01H00000000000000000000000', 'root', 'root@controlplane.local', 'System Root', '$argon2id$v=19$m=65536,t=3,p=4$vQ82Kj2qIfZf5Cq+7N3fXQ$rMjUGEo2eK4JbY/gQf5A019X8k+o2rS6GjXz3v12V0E', 'active', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Assign Admin role to Root user
INSERT INTO iam.user_roles (user_id, role_id, created_at) VALUES 
('01H00000000000000000000000', '01H00000000000000000000002', NOW())
ON CONFLICT DO NOTHING;
