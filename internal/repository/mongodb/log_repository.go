package mongodb

import (
	"context"
	"example.com/taskservice/internal/domain/log"
	"go.mongodb.org/mongo-driver/mongo"
)

type LogRepository struct {
	coll *mongo.Collection
}

func NewLogRepository(db *mongo.Database) *LogRepository {
	return &LogRepository{coll: db.Collection("action_logs")}
}

func (r *LogRepository) Insert(ctx context.Context, entry *log.ActionLog) error {
	_, err := r.coll.InsertOne(ctx, entry)
	return err
}
