package postgres

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type taskScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, scheduled_at, recurrence_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, status, scheduled_at, recurrence_id, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status,
		task.ScheduledAt, task.RecurrenceID,
		task.CreatedAt, task.UpdatedAt,
	)
	return scanTask(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, scheduled_at, recurrence_id, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	if task.RecurrenceID != nil && *task.RecurrenceID > 0 {
		rec, err := r.GetRecurrenceByID(ctx, *task.RecurrenceID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		task.Recurrence = rec
	}

	return task, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
        UPDATE tasks
        SET title = $1,
            description = $2,
            status = $3,
            scheduled_at = $4,
            recurrence_id = $5,
            updated_at = $6
        WHERE id = $7
        RETURNING id, title, description, status, scheduled_at, recurrence_id, created_at, updated_at
    `
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status,
		task.ScheduledAt, task.RecurrenceID,
		task.UpdatedAt, task.ID,
	)
	updated, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	if updated.RecurrenceID != nil && *updated.RecurrenceID > 0 {
		rec, err := r.GetRecurrenceByID(ctx, *task.RecurrenceID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		updated.Recurrence = rec
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {

	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateRecurrence(ctx context.Context, rec *taskdomain.Recurrence) (*taskdomain.Recurrence, error) {
	const query = `
		INSERT INTO task_recurrences (task_id, recurrence_type, interval_days, month_day, specific_dates, odd_even, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, task_id, recurrence_type, interval_days, month_day, specific_dates, odd_even, start_date, end_date, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query,
		rec.TaskID, rec.Type,
		rec.IntervalDays, rec.MonthDay, rec.SpecificDates, rec.OddEven,
		rec.StartDate, rec.EndDate,
		rec.CreatedAt, rec.UpdatedAt,
	)
	return scanRecurrence(row)
}

func (r *Repository) ListActiveRecurrences(ctx context.Context, now time.Time) ([]taskdomain.Recurrence, error) {
	const query = `
		SELECT id, task_id, recurrence_type, interval_days, month_day, specific_dates, odd_even, start_date, end_date, created_at, updated_at
		FROM task_recurrences
		WHERE end_date IS NULL OR end_date >= $1
	`

	rows, err := r.pool.Query(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []taskdomain.Recurrence
	for rows.Next() {
		rec, err := scanRecurrence(rows)
		if err != nil {
			return nil, err
		}
		recs = append(recs, *rec)
	}
	return recs, rows.Err()
}

func (r *Repository) TaskExistsForRecurrenceAndDate(ctx context.Context, recurrenceID int64, scheduledDate time.Time) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM tasks
			WHERE recurrence_id = $1 AND scheduled_at::date = $2::date
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, recurrenceID, scheduledDate).Scan(&exists)
	return exists, err
}

func (r *Repository) UpdateTaskScheduledAt(ctx context.Context, taskID int64, scheduledAt time.Time) error {
	const query = `UPDATE tasks SET scheduled_at = $1, updated_at = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, scheduledAt, time.Now().UTC(), taskID)
	return err
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var task taskdomain.Task
	var status string
	var scheduledAt *time.Time
	var recurrenceID *int64

	err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&scheduledAt,
		&recurrenceID,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	log.Printf("[DEBUG] scanTask: ID=%d, recurrenceID=%v", task.ID, recurrenceID)
	if err != nil {
		return nil, err
	}
	task.Status = taskdomain.Status(status)
	task.ScheduledAt = scheduledAt
	task.RecurrenceID = recurrenceID
	return &task, nil
}

func scanRecurrence(row pgx.Row) (*taskdomain.Recurrence, error) {
	var r taskdomain.Recurrence
	var intervalDays *int
	var monthDay *int
	var specificDates []string
	var oddEven *string
	var endDate *time.Time

	err := row.Scan(
		&r.ID,
		&r.TaskID,
		&r.Type,
		&intervalDays,
		&monthDay,
		&specificDates,
		&oddEven,
		&r.StartDate,
		&endDate,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.IntervalDays = intervalDays
	r.MonthDay = monthDay
	r.SpecificDates = specificDates
	r.OddEven = oddEven
	r.EndDate = endDate
	return &r, nil
}

func (r *Repository) FindOverdueNotNotified(ctx context.Context, now time.Time) ([]*taskdomain.Task, error) {
	const query = `
        SELECT id, title, description, status, scheduled_at, recurrence_id,
               duration_minutes, immutable, overdue_notified, created_at, updated_at
        FROM tasks
        WHERE scheduled_at < $1
          AND status != $2
          AND overdue_notified = false
        ORDER BY scheduled_at
    `
	rows, err := r.pool.Query(ctx, query, now, taskdomain.StatusDone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*taskdomain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *Repository) GetRecurrenceByTaskID(ctx context.Context, taskID int64) (*taskdomain.Recurrence, error) {
	const query = `
		SELECT id, task_id, recurrence_type, interval_days, month_day, specific_dates, odd_even, start_date, end_date, created_at, updated_at
		FROM task_recurrences
		WHERE task_id = $1
	`
	log.Printf("[DEBUG] GetRecurrenceByTaskID: querying task_id=%d", taskID)
	row := r.pool.QueryRow(ctx, query, taskID)
	rec, err := scanRecurrence(row)
	if err != nil {
		log.Printf("[DEBUG] GetRecurrenceByTaskID: error scanning: %v", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	log.Printf("[DEBUG] GetRecurrenceByTaskID: success, rec=%+v", rec)
	return rec, nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
        SELECT id, title, description, status, scheduled_at, recurrence_id, created_at, updated_at
        FROM tasks
        ORDER BY id DESC
    `
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		if task.RecurrenceID != nil && *task.RecurrenceID > 0 {
			log.Printf("[DEBUG] List: loading recurrence for task %d, recurrenceID=%d", task.ID, *task.RecurrenceID)
			rec, err := r.GetRecurrenceByID(ctx, *task.RecurrenceID)
			if err != nil {
				log.Printf("[DEBUG] List: error loading recurrence for task %d: %v", task.ID, err)
				if !errors.Is(err, pgx.ErrNoRows) {
					return nil, err
				}
			}
			if rec != nil {
				task.Recurrence = rec
				log.Printf("[DEBUG] List: loaded recurrence for task %d: %+v", task.ID, rec)
			} else {
				log.Printf("[DEBUG] List: recurrence not found for task %d", task.ID)
			}
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func (r *Repository) GetRecurrenceByID(ctx context.Context, id int64) (*taskdomain.Recurrence, error) {
	const query = `
        SELECT id, task_id, recurrence_type, interval_days, month_day, specific_dates, odd_even, start_date, end_date, created_at, updated_at
        FROM task_recurrences
        WHERE id = $1
    `
	row := r.pool.QueryRow(ctx, query, id)
	rec, err := scanRecurrence(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rec, nil
}
