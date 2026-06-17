package config

import (
	"context"
	"log"
	"os"

	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client
var AsynqServer *asynq.Server

// InitAsynq connects to Redis and initializes the global Asynq client and server.
func InitAsynq() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("[Asynq] REDIS_URL not set in environment")
	}

	// Parse the connection URL
	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		log.Fatalf("[Asynq] Failed to parse REDIS_URL: %v", err)
	}

	// Initialize the client (used by handlers to enqueue tasks)
	AsynqClient = asynq.NewClient(redisOpt)

	// Initialize the server (used by workers to process tasks)
	AsynqServer = asynq.NewServer(
		redisOpt,
		asynq.Config{
			// Process up to 10 jobs concurrently
			Concurrency: 10,
			// Custom error handler could be added here
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("[Asynq Error] Task %s failed: %v", task.Type(), err)
			}),
		},
	)

	log.Println("[Asynq] Client and Server initialized ✓")
}
