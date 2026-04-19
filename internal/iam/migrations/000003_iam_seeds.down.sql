DELETE FROM iam.user_roles
WHERE user_id IN (
	SELECT id FROM iam.users WHERE username = 'root'
)
OR role_id IN (
	SELECT id FROM iam.roles WHERE name IN ('user', 'root')
);

DELETE FROM iam.role_permissions
WHERE role_id IN (
	SELECT id FROM iam.roles WHERE name IN ('user', 'root')
)
OR permission_id IN (
	SELECT id FROM iam.permissions WHERE slug IN (
		'iam:user:read',
		'iam:user:write',
		'iam:role:read',
		'iam:role:assign'
	)
);

DELETE FROM iam.users WHERE username = 'root';
DELETE FROM iam.permissions WHERE slug IN ('iam:user:read', 'iam:user:write', 'iam:role:read', 'iam:role:assign');
DELETE FROM iam.roles WHERE name IN ('user', 'root');
