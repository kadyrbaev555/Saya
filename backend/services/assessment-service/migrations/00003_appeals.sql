-- +goose Up
CREATE TABLE appeals (
    user_id uuid PRIMARY KEY,
    id uuid NOT NULL UNIQUE,
    assessment_id uuid NOT NULL REFERENCES assessments(id),
    description text NOT NULL CHECK (char_length(description) BETWEEN 1 AND 1200),
    state text NOT NULL CHECK (state IN ('pending','published','withdrawn','declined')),
    reported boolean NOT NULL DEFAULT false
);
CREATE VIEW eligible_appeals AS
SELECT a.* FROM appeals a
JOIN LATERAL (
    SELECT id,result FROM assessments WHERE user_id=a.user_id AND completed_at IS NOT NULL
    ORDER BY completed_at DESC,id DESC LIMIT 1
) latest ON latest.id=a.assessment_id
WHERE latest.result->'directions' ? 'financial';

CREATE TABLE payout_bindings (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL UNIQUE,
    pending_id uuid NOT NULL UNIQUE,
    card_token text
);

CREATE TABLE test_donations (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    appeal_id uuid NOT NULL,
    recipient_id uuid NOT NULL,
    recipient_token text NOT NULL,
    amount integer NOT NULL CHECK (amount BETWEEN 100 AND 100000),
    provider_id text UNIQUE,
    payout_state text NOT NULL DEFAULT 'queued' CHECK (payout_state IN ('queued','checking','paid','declined')),
    payout_due timestamptz NOT NULL DEFAULT now(),
    payout_id text UNIQUE,
    succeeded boolean
);

-- +goose Down
DROP TABLE test_donations;
DROP TABLE payout_bindings;
DROP VIEW eligible_appeals;
DROP TABLE appeals;
