UPDATE iam.users
SET security_level = 100,
    updated_at = NOW()
WHERE username = 'root';
