-- Insert initial roles
INSERT INTO roles (id, name) VALUES
    ('00000000-0000-0000-0000-000000000001', 'user'),
    ('00000000-0000-0000-0000-000000000002', 'admin');

-- Insert admin user (example email and password hash)
INSERT INTO users (id, email, password_hash) VALUES
    ('00000000-0000-0000-0000-000000000010', 'admin@example.com', 'admin_password_hash');

-- Assign admin role to admin user
INSERT INTO user_roles (user_id, role_id) VALUES
    ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000002');

-- Insert example permissions
INSERT INTO permissions (id, resource, action) VALUES
    ('00000000-0000-0000-0000-000000000101', 'task', 'read'),
    ('00000000-0000-0000-0000-000000000102', 'task', 'write'),
    ('00000000-0000-0000-0000-000000000103', 'user', 'read'),
    ('00000000-0000-0000-0000-000000000104', 'user', 'write');

-- Map permissions to roles
-- User role: can read tasks
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000101');

-- Admin role: can read/write tasks and users
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000101'),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000102'),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000103'),
    ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000104');
