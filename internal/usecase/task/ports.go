package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
	CreateRecurrence(ctx context.Context, rec *taskdomain.Recurrence) (*taskdomain.Recurrence, error)
	ListActiveRecurrences(ctx context.Context, now time.Time) ([]taskdomain.Recurrence, error)
	TaskExistsForRecurrenceAndDate(ctx context.Context, recurrenceID int64, scheduledDate time.Time) (bool, error)
	UpdateTaskScheduledAt(ctx context.Context, taskID int64, scheduledAt time.Time) error
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
	GenerateUpcomingTasks(ctx context.Context, daysAhead int) error
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	ScheduledAt *time.Time
	Recurrence  *RecurrenceInput
}

type RecurrenceInput struct {
	Type          taskdomain.RecurrenceType
	IntervalDays  *int
	MonthDay      *int
	SpecificDates []string
	OddEven       *string
	StartDate     time.Time
	EndDate       *time.Time
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	ScheduledAt *time.Time
}
