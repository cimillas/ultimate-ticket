-- Add hold ownership and scope idempotency per user.
ALTER TABLE holds ADD COLUMN IF NOT EXISTS user_id UUID;

DO $$
DECLARE
	legacy_user_id UUID;
	has_legacy_holds BOOLEAN;
BEGIN
	SELECT EXISTS (SELECT 1 FROM holds WHERE user_id IS NULL) INTO has_legacy_holds;
	IF NOT has_legacy_holds THEN
		RETURN;
	END IF;

	SELECT id INTO legacy_user_id FROM users ORDER BY created_at ASC LIMIT 1;
	IF legacy_user_id IS NULL THEN
		legacy_user_id := gen_random_uuid();
		INSERT INTO users (id, username, email, password_hash, role, created_at, updated_at)
		VALUES (legacy_user_id, 'legacy-holds', 'legacy-holds@local', 'legacy', 'user', NOW(), NOW());
	END IF;

	UPDATE holds SET user_id = legacy_user_id WHERE user_id IS NULL;
END $$;

ALTER TABLE holds ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE holds DROP CONSTRAINT IF EXISTS holds_user_id_fkey;
ALTER TABLE holds ADD CONSTRAINT holds_user_id_fkey
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS holds_idempotency_unique;
CREATE UNIQUE INDEX IF NOT EXISTS holds_idempotency_unique
	ON holds(event_id, zone_id, user_id, idempotency_key);
CREATE INDEX IF NOT EXISTS holds_user_id_idx ON holds(user_id);
