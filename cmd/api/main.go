package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	mongodbrepo "example.com/taskservice/internal/repository/mongodb"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	transporthttp "example.com/taskservice/internal/transport/http"
	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
	"example.com/taskservice/internal/usecase/task"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := infrastructurepostgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		logger.Error("open postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		logger.Error("connect mongodb", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			logger.Error("disconnect mongodb", "error", err)
		}
	}()
	mongoDB := mongoClient.Database(cfg.MongoDB)
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "task_id", Value: 1}, {Key: "timestamp", Value: -1}},
	}
	_, err = mongoDB.Collection("action_logs").Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		logger.Warn("failed to create mongodb index", "error", err)
	}
	logRepo := mongodbrepo.NewLogRepository(mongoDB)

	taskRepo := postgresrepo.New(pool)
	taskUsecase := task.NewService(taskRepo, logRepo)

	taskHandler := httphandlers.NewTaskHandler(taskUsecase)
	docsHandler := swaggerdocs.NewHandler()
	router := transporthttp.NewRouter(taskHandler, docsHandler)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go runRecurrenceGenerator(ctx, taskUsecase, cfg.RecurrenceGenInterval, logger)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown http server", "error", err)
		}
	}()

	logger.Info("http server started", "addr", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("listen and serve", "error", err)
		os.Exit(1)
	}
}

func runRecurrenceGenerator(ctx context.Context, usecase task.Usecase, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := usecase.GenerateUpcomingTasks(ctx, 7); err != nil {
				logger.Error("generate upcoming tasks", "error", err)
			}
		}
	}
}

type config struct {
	HTTPAddr              string
	DatabaseDSN           string
	MongoURI              string
	MongoDB               string
	RecurrenceGenInterval time.Duration
}

func loadConfig() config {
	return config{
		HTTPAddr:              getEnv("HTTP_ADDR", ":8080"),
		DatabaseDSN:           getEnv("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/taskservice?sslmode=disable"),
		MongoURI:              getEnv("MONGO_URI", "mongodb://admin:adminpass@localhost:27017"),
		MongoDB:               getEnv("MONGO_DB", "taskservice_logs"),
		RecurrenceGenInterval: getEnvAsDuration("RECURRENCE_GEN_INTERVAL", 1*time.Hour),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
