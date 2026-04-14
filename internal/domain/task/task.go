package task

import "time"

type Status string

const (
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
	StatusRecurring  Status = "recurring"
)

type RecurrenceType string

const (
	RecurrenceDaily    RecurrenceType = "daily"
	RecurrenceMonthly  RecurrenceType = "monthly"
	RecurrenceSpecific RecurrenceType = "specific_dates"
	RecurrenceOddEven  RecurrenceType = "odd_even"
)

type Recurrence struct {
	ID            int64          `json:"id"`
	TaskID        int64          `json:"task_id"`
	Type          RecurrenceType `json:"type"`
	IntervalDays  *int           `json:"interval_days,omitempty"`
	MonthDay      *int           `json:"month_day,omitempty"`
	SpecificDates []string       `json:"specific_dates,omitempty"`
	OddEven       *string        `json:"odd_even,omitempty"`
	StartDate     time.Time      `json:"start_date"`
	EndDate       *time.Time     `json:"end_date,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Task struct {
	ID           int64       `json:"id"`
	Title        string      `json:"title"`
	Description  string      `json:"description"`
	Status       Status      `json:"status"`
	ScheduledAt  *time.Time  `json:"scheduled_at,omitempty"`
	RecurrenceID *int64      `json:"recurrence_id,omitempty"`
	Recurrence   *Recurrence `json:"recurrence,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusInProgress, StatusDone, StatusCancelled, StatusRecurring:
		return true
	default:
		return false
	}
}
