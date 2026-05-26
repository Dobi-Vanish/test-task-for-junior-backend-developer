package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	logdomain "example.com/taskservice/internal/domain/log"
	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo    Repository
	logRepo LogRepository
	now     func() time.Time
}

type LogRepository interface {
	Insert(ctx context.Context, entry *logdomain.ActionLog) error
}

func NewService(repo Repository, logRepo LogRepository) *Service {
	return &Service{
		repo:    repo,
		logRepo: logRepo,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		ScheduledAt: normalized.ScheduledAt,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	if input.Recurrence != nil {
		rec := &taskdomain.Recurrence{
			TaskID:        created.ID,
			Type:          input.Recurrence.Type,
			IntervalDays:  input.Recurrence.IntervalDays,
			MonthDay:      input.Recurrence.MonthDay,
			SpecificDates: input.Recurrence.SpecificDates,
			OddEven:       input.Recurrence.OddEven,
			StartDate:     input.Recurrence.StartDate,
			EndDate:       input.Recurrence.EndDate,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		rec, err = s.repo.CreateRecurrence(ctx, rec)
		if err != nil {
			_ = s.repo.Delete(ctx, created.ID)
			return nil, FailedToCreateRecurrence
		}
		created.RecurrenceID = &rec.ID
		created, err = s.repo.Update(ctx, created)
		if err != nil {
			return nil, err
		}

		created.Recurrence = rec
	}

	_ = s.logRepo.Insert(ctx, &logdomain.ActionLog{
		TaskID:    created.ID,
		Action:    logdomain.ActionCreated,
		Timestamp: s.now(),
		Details:   fmt.Sprintf("Task '%s' created", created.Title),
	})

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, IdMustBePositive
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, IdMustBePositive
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:           id,
		Title:        normalized.Title,
		Description:  normalized.Description,
		Status:       normalized.Status,
		ScheduledAt:  normalized.ScheduledAt,
		RecurrenceID: existing.RecurrenceID,
		UpdatedAt:    s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	if model.Status == taskdomain.StatusDone && existing.Status != taskdomain.StatusDone {
		_ = s.logRepo.Insert(ctx, &logdomain.ActionLog{
			TaskID:    updated.ID,
			Action:    logdomain.ActionCompleted,
			Timestamp: s.now(),
			Details:   fmt.Sprintf("Task '%s' completed", updated.Title),
		})
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return IdMustBePositive
	}
	task, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	_ = s.logRepo.Insert(ctx, &logdomain.ActionLog{
		TaskID:    task.ID,
		Action:    "deleted",
		Timestamp: s.now(),
		Details:   fmt.Sprintf("Task '%s' deleted", task.Title),
	})
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func (s *Service) GenerateUpcomingTasks(ctx context.Context, daysAhead int) error {
	now := s.now().Truncate(24 * time.Hour)
	endDate := now.AddDate(0, 0, daysAhead)

	recs, err := s.repo.ListActiveRecurrences(ctx, now)
	if err != nil {
		return err
	}

	for _, rec := range recs {
		dates, err := generateDatesForRecurrence(rec, now, endDate)
		if err != nil {
			continue
		}
		for _, date := range dates {
			exists, err := s.repo.TaskExistsForRecurrenceAndDate(ctx, rec.ID, date)
			if err != nil || exists {
				continue
			}
			parent, err := s.repo.GetByID(ctx, rec.TaskID)
			if err != nil {
				continue
			}
			task := &taskdomain.Task{
				Title:        parent.Title,
				Description:  parent.Description,
				Status:       taskdomain.StatusInProgress,
				ScheduledAt:  &date,
				RecurrenceID: &rec.ID,
				CreatedAt:    s.now(),
				UpdatedAt:    s.now(),
			}
			_, _ = s.repo.Create(ctx, task)
		}
	}
	return nil
}

func generateDatesForRecurrence(rec taskdomain.Recurrence, from, to time.Time) ([]time.Time, error) {
	var dates []time.Time
	current := from
	for !current.After(to) {
		if current.Before(rec.StartDate) {
			current = current.AddDate(0, 0, 1)
			continue
		}
		if rec.EndDate != nil && current.After(*rec.EndDate) {
			break
		}
		match := false
		switch rec.Type {
		case taskdomain.RecurrenceDaily:
			interval := 1
			if rec.IntervalDays != nil && *rec.IntervalDays > 0 {
				interval = *rec.IntervalDays
			}
			daysDiff := int(current.Sub(rec.StartDate).Hours() / 24)
			if daysDiff%interval == 0 {
				match = true
			}
		case taskdomain.RecurrenceMonthly:
			if rec.MonthDay != nil && current.Day() == *rec.MonthDay {
				match = true
			}
		case taskdomain.RecurrenceSpecific:
			dateStr := current.Format("2006-01-02")
			for _, d := range rec.SpecificDates {
				if d == dateStr {
					match = true
					break
				}
			}
		case taskdomain.RecurrenceOddEven:
			if rec.OddEven != nil {
				isOdd := current.Day()%2 != 0
				if (*rec.OddEven == "odd" && isOdd) || (*rec.OddEven == "even" && !isOdd) {
					match = true
				}
			}
		}
		if match {
			dates = append(dates, current)
		}
		current = current.AddDate(0, 0, 1)
	}
	return dates, nil
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" {
		return CreateInput{}, TitleIsRequired
	}
	if input.Status == "" {
		input.Status = taskdomain.StatusInProgress
	}
	if !input.Status.Valid() {
		return CreateInput{}, InvalidStatus
	}
	if input.Recurrence != nil {
		if err := validateRecurrenceInput(*input.Recurrence); err != nil {
			return CreateInput{}, err
		}
	}
	return input, nil
}

func validateRecurrenceInput(ri RecurrenceInput) error {
	switch ri.Type {
	case taskdomain.RecurrenceDaily:
		if ri.IntervalDays != nil && *ri.IntervalDays <= 0 {
			return IntervalDaysPositive
		}
	case taskdomain.RecurrenceMonthly:
		if ri.MonthDay == nil || *ri.MonthDay < 1 || *ri.MonthDay > 31 {
			return MonthDay
		}
	case taskdomain.RecurrenceSpecific:
		if len(ri.SpecificDates) == 0 {
			return SpecificDatesEmpty
		}
	case taskdomain.RecurrenceOddEven:
		if ri.OddEven == nil || (*ri.OddEven != "odd" && *ri.OddEven != "even") {
			return OddEven
		}
	default:
		return UnknownRecurrence
	}
	if ri.StartDate.IsZero() {
		return StartDate
	}
	if ri.EndDate != nil && !ri.EndDate.After(ri.StartDate) {
		return StartDateAfterEndDate
	}
	return nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" {
		return UpdateInput{}, TitleIsRequired
	}
	if !input.Status.Valid() {
		return UpdateInput{}, InvalidStatus
	}
	return input, nil
}
