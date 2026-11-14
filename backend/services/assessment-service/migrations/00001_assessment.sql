-- +goose Up
CREATE TABLE assessments(id UUID PRIMARY KEY,user_id UUID NOT NULL,input JSONB NOT NULL,result JSONB NOT NULL,status TEXT NOT NULL CHECK(status IN('pending','completed','failed')),created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE INDEX assessments_owner ON assessments(user_id,created_at DESC);
CREATE TABLE outbox(id UUID PRIMARY KEY,payload JSONB NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE notifications(id UUID PRIMARY KEY,user_id UUID NOT NULL,status TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now());
-- +goose Down
DROP TABLE notifications;
DROP TABLE outbox;
DROP TABLE assessments;
