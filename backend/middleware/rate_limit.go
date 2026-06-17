package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
)

func RateLimit(prefix string, keyFn func(c *gin.Context) string, max int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		key := fmt.Sprintf("%s:%s", prefix, keyFn(c))

		count, err := config.RDB.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			config.RDB.Expire(ctx, key, window)
		}

		if count > int64(max) {
			ttl, _ := config.RDB.TTL(ctx, key).Result()
			c.Header("Retry-After", fmt.Sprintf("%.0f", ttl.Seconds()))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "too many requests",
				"retry_after": fmt.Sprintf("%.0f seconds", ttl.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func ByIP(c *gin.Context) string {
	return c.ClientIP()
}

func ByBodyEmail(c *gin.Context) string {
	return c.ClientIP()
}
