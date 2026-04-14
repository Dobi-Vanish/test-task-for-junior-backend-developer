package task

import "errors"

var (
	IdMustBePositive         = errors.New("ID must be positive")
	ErrInvalidInput          = errors.New("invalid task input")
	TitleIsRequired          = errors.New("title is required")
	InvalidStatus            = errors.New("invalid status")
	IntervalDaysPositive     = errors.New("interval days must be positive")
	MonthDay                 = errors.New("month day must be between 1 and 31")
	SpecificDatesEmpty       = errors.New("specific dates must not be empty")
	UnknownRecurrence        = errors.New("unknown recurrence type")
	OddEven                  = errors.New("odd even must be 'odd' or 'even'")
	StartDate                = errors.New("start day is required")
	StartDateAfterEndDate    = errors.New("end date must be after start date")
	FailedToCreateRecurrence = errors.New("failed to create recurrence")
)
