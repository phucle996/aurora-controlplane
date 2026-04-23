DROP TRIGGER IF EXISTS set_timestamp_secret_key_versions ON core.secret_key_versions;

DROP INDEX IF EXISTS core.idx_core_secret_key_versions_family_rotated_at;
DROP INDEX IF EXISTS core.idx_core_secret_key_versions_family_state;

DROP TABLE IF EXISTS core.secret_key_versions;
