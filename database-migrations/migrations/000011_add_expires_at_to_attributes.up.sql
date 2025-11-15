ALTER TABLE user_attributes ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP;

ALTER TABLE resource_attributes ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP;

ALTER TABLE resource_attributes ADD COLUMN IF NOT EXISTS source VARCHAR(50) DEFAULT 'system';

CREATE INDEX IF NOT EXISTS idx_user_attributes_expires_at ON user_attributes(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_resource_attributes_expires_at ON resource_attributes(expires_at) WHERE expires_at IS NOT NULL;
