package log

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Action string

const (
	ActionCreated   Action = "created"
	ActionCompleted Action = "completed"
	ActionDeleted   Action = "deleted"
)

type ActionLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	TaskID    int64              `bson:"task_id"`
	Action    Action             `bson:"action"`
	Timestamp time.Time          `bson:"timestamp"`
	Details   string             `bson:"details"`
}
