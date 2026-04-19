UPDATE iam.users
SET security_level = 0,
    updated_at = NOW()
WHERE username = 'root';
