DROP INDEX IF EXISTS idx_user_role_unique;
DROP INDEX IF EXISTS idx_role_permission_unique;
DROP INDEX IF EXISTS idx_audit_logs_decision;
DROP INDEX IF EXISTS idx_audit_logs_timestamp;
DROP INDEX IF EXISTS idx_audit_logs_resource;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_resource_attributes_type;
DROP INDEX IF EXISTS idx_resource_attributes_lookup;
DROP INDEX IF EXISTS idx_user_attributes_key;
DROP INDEX IF EXISTS idx_user_attributes_user_id;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS resource_attributes;
DROP TABLE IF EXISTS user_attributes;

ALTER TABLE role_permissions DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS updated_at;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS created_at;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS granted_at;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS granted_by;
ALTER TABLE role_permissions DROP COLUMN IF EXISTS id;

ALTER TABLE user_roles DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE user_roles DROP COLUMN IF EXISTS updated_at;
ALTER TABLE user_roles DROP COLUMN IF EXISTS created_at;
ALTER TABLE user_roles DROP COLUMN IF EXISTS expires_at;
ALTER TABLE user_roles DROP COLUMN IF EXISTS assigned_at;
ALTER TABLE user_roles DROP COLUMN IF EXISTS assigned_by;
ALTER TABLE user_roles DROP COLUMN IF EXISTS id;

ALTER TABLE permissions DROP COLUMN IF EXISTS description;
ALTER TABLE permissions DROP COLUMN IF EXISTS conditions;
ALTER TABLE permissions DROP COLUMN IF EXISTS scope;

ALTER TABLE roles DROP COLUMN IF EXISTS description;

ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
ALTER TABLE users DROP COLUMN IF EXISTS position;
ALTER TABLE users DROP COLUMN IF EXISTS department;
ALTER TABLE users DROP COLUMN IF EXISTS last_name;
ALTER TABLE users DROP COLUMN IF EXISTS first_name;