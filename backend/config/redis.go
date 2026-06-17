package config

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func ConnectRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("[Redis] REDIS_URL not set in environment")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("[Redis] Failed to parse REDIS_URL: %v", err)
	}

	RDB = redis.NewClient(opts)

	// Ping to verify connection
	ctx := context.Background()
	if _, err := RDB.Ping(ctx).Result(); err != nil {
		log.Fatalf("[Redis] Failed to connect: %v", err)
	}

	log.Println("[Redis] Connected ✓")
}
