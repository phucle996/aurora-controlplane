DROP TABLE IF EXISTS iam.device_challenges;

DROP TRIGGER IF EXISTS set_timestamp_roles ON iam.roles;

ALTER TABLE iam.devices
	DROP COLUMN IF EXISTS revoked_at,
	DROP COLUMN IF EXISTS is_suspicious;

ALTER TABLE iam.permissions
	DROP CONSTRAINT IF EXISTS permissions_name_key;

UPDATE iam.permissions
SET slug = name
WHERE (slug IS NULL OR BTRIM(slug) = '')
  AND name IS NOT NULL;

ALTER TABLE iam.permissions
	ALTER COLUMN slug SET NOT NULL;

ALTER TABLE iam.permissions
	DROP COLUMN IF EXISTS name;

ALTER TABLE iam.roles
	DROP COLUMN IF EXISTS updated_at,
	DROP COLUMN IF EXISTS level;
