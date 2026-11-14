-- +goose Up
CREATE TABLE users(id UUID PRIMARY KEY, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'USER' CHECK(role IN ('USER','ADMIN')), created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE sessions(id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id), expires_at TIMESTAMPTZ NOT NULL, revoked BOOLEAN NOT NULL DEFAULT false);
CREATE TABLE refresh_tokens(hash TEXT PRIMARY KEY, session_id UUID NOT NULL REFERENCES sessions(id), consumed BOOLEAN NOT NULL DEFAULT false);
CREATE INDEX sessions_user ON sessions(user_id);
CREATE INDEX refresh_session ON refresh_tokens(session_id);
-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE sessions;
DROP TABLE users;
