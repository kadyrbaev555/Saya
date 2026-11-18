-- +goose Up
ALTER TABLE users ADD COLUMN full_name text, ADD COLUMN phone text;

CREATE TABLE pending_registrations (
    id uuid PRIMARY KEY,
    email text NOT NULL UNIQUE,
    full_name text NOT NULL,
    phone text NOT NULL,
    password_hash text NOT NULL,
    code_digest bytea NOT NULL,
    sent_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL DEFAULT now() + interval '10 minutes',
    attempts_left integer NOT NULL DEFAULT 5 CHECK (attempts_left BETWEEN 0 AND 5)
);
CREATE INDEX pending_registrations_expiry ON pending_registrations(expires_at);

-- +goose Down
DROP TABLE pending_registrations;
ALTER TABLE users DROP COLUMN phone, DROP COLUMN full_name;
