ALTER TABLE iam.roles
	ADD COLUMN IF NOT EXISTS level INTEGER NOT NULL DEFAULT 100,
	ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE iam.roles
SET level = CASE
	WHEN name = 'root' THEN 0
	WHEN name = 'user' THEN 100
	ELSE level
END;

DROP TRIGGER IF EXISTS set_timestamp_roles ON iam.roles;
CREATE TRIGGER set_timestamp_roles
BEFORE UPDATE ON iam.roles
FOR EACH ROW
EXECUTE FUNCTION iam.trigger_set_timestamp();

ALTER TABLE iam.permissions
	ADD COLUMN IF NOT EXISTS name VARCHAR(100);

UPDATE iam.permissions
SET name = slug
WHERE name IS NULL OR BTRIM(name) = '';

ALTER TABLE iam.permissions
	ALTER COLUMN name SET NOT NULL;

ALTER TABLE iam.permissions
	ALTER COLUMN slug DROP NOT NULL;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'permissions_name_key'
		  AND conrelid = 'iam.permissions'::regclass
	) THEN
		ALTER TABLE iam.permissions
			ADD CONSTRAINT permissions_name_key UNIQUE (name);
	END IF;
END $$;

ALTER TABLE iam.devices
	ADD COLUMN IF NOT EXISTS is_suspicious BOOLEAN NOT NULL DEFAULT FALSE,
	ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS iam.device_challenges (
	id VARCHAR(26) PRIMARY KEY,
	device_id VARCHAR(26) NOT NULL UNIQUE REFERENCES iam.devices(id) ON DELETE CASCADE,
	user_id VARCHAR(26) NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
	nonce TEXT NOT NULL,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ DEFAULT NOW()
);
