package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client
var AsynqServer *asynq.Server

func InitAsynq() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("[Asynq] REDIS_URL not set in environment")
	}

	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		log.Fatalf("[Asynq] Failed to parse REDIS_URL: %v", err)
	}

	AsynqClient = asynq.NewClient(redisOpt)

	AsynqServer = asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency:              10,
			DelayedTaskCheckInterval: 30 * time.Second,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("[Asynq Error] Task %s failed: %v", task.Type(), err)
			}),
		},
	)

	log.Println("[Asynq] Client and Server initialized ✓")
}
