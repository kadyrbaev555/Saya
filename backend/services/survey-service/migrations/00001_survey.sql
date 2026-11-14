-- +goose Up
CREATE TABLE submissions(id UUID PRIMARY KEY,user_id UUID NOT NULL,idempotency_key UUID NOT NULL,payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),UNIQUE(user_id,idempotency_key));
CREATE INDEX submissions_owner ON submissions(user_id,created_at DESC);
CREATE TABLE outbox(id UUID PRIMARY KEY,payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now());
-- +goose Down
DROP TABLE outbox;
DROP TABLE submissions;
