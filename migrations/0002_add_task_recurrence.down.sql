DROP TABLE IF EXISTS task_recurrences;
DROP TYPE IF EXISTS recurrence_type;
ALTER TABLE tasks DROP COLUMN IF EXISTS recurrence_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS scheduled_at;