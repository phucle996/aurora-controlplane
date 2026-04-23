DO $$
DECLARE
	current_def TEXT;
BEGIN
	SELECT pg_get_constraintdef(c.oid)
	INTO current_def
	FROM pg_constraint c
	WHERE c.conrelid = 'core.workspaces'::regclass
	  AND c.conname = 'workspaces_tenant_id_fkey';

	IF current_def IS NOT NULL AND POSITION('ON DELETE SET NULL' IN current_def) > 0 THEN
		RETURN;
	END IF;

	ALTER TABLE core.workspaces
		DROP CONSTRAINT IF EXISTS workspaces_tenant_id_fkey;

	ALTER TABLE core.workspaces
		ADD CONSTRAINT workspaces_tenant_id_fkey
		FOREIGN KEY (tenant_id) REFERENCES core.tenants(id) ON DELETE SET NULL;
END $$;
