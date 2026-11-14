-- +goose Up
CREATE TABLE demo_inbox(event_id UUID PRIMARY KEY, completed_at TIMESTAMPTZ NOT NULL DEFAULT now());
-- +goose Down
DROP TABLE demo_inbox;
