-- +goose Up
ALTER TABLE assessments ADD COLUMN completed_at timestamptz;
-- Earlier records have no completion timestamp; use the best recorded time.
UPDATE assessments a SET completed_at=COALESCE(
  (SELECT n.created_at FROM notifications n WHERE n.id=a.id AND n.status='completed'),a.created_at)
WHERE a.status='completed';
CREATE INDEX assessment_history ON assessments(user_id,completed_at DESC,id DESC) WHERE completed_at IS NOT NULL;
CREATE TABLE weekly_reminders (
 user_id uuid PRIMARY KEY, assessment_id uuid NOT NULL REFERENCES assessments(id),
 due_at timestamptz NOT NULL, announced boolean NOT NULL DEFAULT false
);
CREATE INDEX weekly_due ON weekly_reminders(due_at) WHERE NOT announced;
INSERT INTO weekly_reminders(user_id,assessment_id,due_at)
SELECT DISTINCT ON(user_id) user_id,id,completed_at+interval '168 hours'
FROM assessments WHERE completed_at IS NOT NULL AND result->>'version'='2'
ORDER BY user_id,completed_at DESC,id DESC;
CREATE TABLE push_devices(
 token text PRIMARY KEY, user_id uuid NOT NULL, enabled_at timestamptz NOT NULL DEFAULT now(),
 last_assessment uuid, retry_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX push_devices_owner ON push_devices(user_id);
-- +goose Down
DROP TABLE push_devices;
DROP TABLE weekly_reminders;
DROP INDEX assessment_history;
ALTER TABLE assessments DROP COLUMN completed_at;
