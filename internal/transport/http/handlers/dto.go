package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	Recurrence  *recurrenceDTO    `json:"recurrence,omitempty"`
}

type recurrenceDTO struct {
	ID            int64    `json:"id,omitempty"`
	TaskID        int64    `json:"task_id,omitempty"`
	Type          string   `json:"type"`
	IntervalDays  *int     `json:"interval_days,omitempty"`
	MonthDay      *int     `json:"month_day,omitempty"`
	SpecificDates []string `json:"specific_dates,omitempty"`
	OddEven       *string  `json:"odd_even,omitempty"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
	CreatedAt     string   `json:"created_at,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

type taskDTO struct {
	ID           int64             `json:"id"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Status       taskdomain.Status `json:"status"`
	ScheduledAt  *time.Time        `json:"scheduled_at,omitempty"`
	RecurrenceID *int64            `json:"recurrence_id,omitempty"`
	Recurrence   *recurrenceDTO    `json:"recurrence"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:           task.ID,
		Title:        task.Title,
		Description:  task.Description,
		Status:       task.Status,
		ScheduledAt:  task.ScheduledAt,
		RecurrenceID: task.RecurrenceID,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
	if task.Recurrence != nil {
		dto.Recurrence = newRecurrenceDTO(task.Recurrence)
	}
	return dto
}

func newRecurrenceDTO(r *taskdomain.Recurrence) *recurrenceDTO {
	if r == nil {
		return nil
	}
	dto := &recurrenceDTO{
		ID:            r.ID,
		TaskID:        r.TaskID,
		Type:          string(r.Type),
		IntervalDays:  r.IntervalDays,
		MonthDay:      r.MonthDay,
		SpecificDates: r.SpecificDates,
		OddEven:       r.OddEven,
		StartDate:     r.StartDate.Format("2006-01-02"),
		CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     r.UpdatedAt.Format(time.RFC3339),
	}
	if !r.StartDate.IsZero() {
		dto.StartDate = r.StartDate.Format("2006-01-02")
	}
	if r.EndDate != nil {
		endDate := r.EndDate.Format("2006-01-02")
		dto.EndDate = &endDate
	}
	return dto
}
