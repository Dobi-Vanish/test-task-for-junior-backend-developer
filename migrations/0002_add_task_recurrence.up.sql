ALTER TABLE tasks ADD COLUMN scheduled_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN recurrence_id BIGINT;

CREATE TYPE recurrence_type AS ENUM ('daily', 'monthly', 'specific_dates', 'odd_even');

CREATE TABLE IF NOT EXISTS task_recurrences (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    recurrence_type recurrence_type NOT NULL,
    interval_days INT NULL,
    month_day INT NULL,
    specific_dates DATE[] NULL,
    odd_even VARCHAR(4) NULL CHECK (odd_even IN ('odd', 'even')),
    start_date DATE NOT NULL,
    end_date DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

ALTER TABLE task_recurrences ALTER COLUMN specific_dates TYPE TEXT[] USING specific_dates::TEXT[];
CREATE INDEX idx_task_recurrences_task_id ON task_recurrences(task_id);
CREATE INDEX idx_tasks_recurrence_id ON tasks(recurrence_id);