package task

import "errors"

var (
	ErrNotFound = errors.New("task not found")
	ErrDsnEmpty = errors.New("database dsn is empty")
	MissingID   = errors.New("missing task id")
	InvalidID   = errors.New("invalid task id")
)
