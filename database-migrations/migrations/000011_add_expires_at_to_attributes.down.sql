DROP INDEX IF EXISTS idx_resource_attributes_expires_at;
DROP INDEX IF EXISTS idx_user_attributes_expires_at;

ALTER TABLE resource_attributes DROP COLUMN IF EXISTS source;

ALTER TABLE resource_attributes DROP COLUMN IF EXISTS expires_at;

ALTER TABLE user_attributes DROP COLUMN IF EXISTS expires_at;
