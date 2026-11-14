-- REVIEW ONLY: do not execute without explicit approval.
-- Connect as saya_owner to database saya_auth. Stop Auth/Gateway during migration.
-- Supply auth_password as a psql variable from the protected local secret source.
-- Keep the bootstrap role (PostgreSQL OID 10) SUPERUSER, but disable its login.
-- Transfer only Auth objects; do not REASSIGN OWNED across the whole cluster.
BEGIN;
ALTER ROLE saya_auth RENAME TO saya_bootstrap;
CREATE ROLE saya_auth LOGIN PASSWORD :'auth_password';
ALTER DATABASE saya_auth OWNER TO saya_auth;
ALTER TABLE users OWNER TO saya_auth;
ALTER TABLE sessions OWNER TO saya_auth;
ALTER TABLE refresh_tokens OWNER TO saya_auth;
ALTER TABLE demo_inbox OWNER TO saya_auth;
ALTER TABLE goose_db_version OWNER TO saya_auth;
ALTER ROLE saya_bootstrap NOLOGIN;
COMMIT;
-- Separately rotate the local saya_owner password and update the ignored .env
-- using protected input, never command arguments or printed credentials.
