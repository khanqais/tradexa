package middleware

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/khanqais/tradexa/config"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = strings.TrimSpace(c.Query("token"))
		}

		if tokenString == "" {
			log.Println("[ERROR] authorization token missing")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token missing"})
			c.Abort()
			return
		}

		// --- Cache-first blacklist check (saves 1 Redis read per request) ---
		blacklistKey := "blacklist:" + tokenString
		if isBlacklisted(blacklistKey, tokenString) {
			log.Println("[ERROR] Token is blacklisted (logged out)")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked, please login again"})
			c.Abort()
			return
		}

		secret := os.Getenv("JWT_SECRET")

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			log.Printf("[ERROR] Token validation failed: err=%v, valid=%v\n", err, token != nil && token.Valid)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["user_id"])
			c.Set("email", claims["email"])
			c.Set("role", claims["role"])
			c.Set("name", claims["name"])
			c.Set("raw_token", tokenString)
		}

		c.Next()
	}
}

// isBlacklisted checks the in-memory cache first, falling back to Redis on a miss.
func isBlacklisted(redisKey, token string) bool {
	if blacklisted, found := blacklistCache.get(token); found {
		return blacklisted
	}
	// Cache miss — ask Redis.
	count, err := config.RDB.Exists(context.Background(), redisKey).Result()
	if err != nil {
		// On Redis error, fail open (don't block legitimate users).
		log.Printf("[Auth] Redis blacklist check failed: %v", err)
		return false
	}
	blacklisted := count > 0
	if blacklisted {
		// Cache positive results for a long time — blacklisted tokens stay blacklisted.
		blacklistCache.set(token, true, 24*time.Hour)
	} else {
		// Cache negative results briefly — allows logout to propagate within ~60 s.
		blacklistCache.set(token, false, notBlacklistedTTL)
	}
	return blacklisted
}

// BlacklistToken writes the token to Redis and immediately invalidates any
// cached "not blacklisted" entry so the logout takes effect right away.
func BlacklistToken(tokenString string, ttl time.Duration) error {
	ctx := context.Background()
	if err := config.RDB.Set(ctx, "blacklist:"+tokenString, "1", ttl).Err(); err != nil {
		return err
	}
	// Force the cache to reflect the new blacklisted state immediately.
	blacklistCache.set(tokenString, true, ttl)
	return nil
}

// InitMiddleware must be called once at startup to launch the cache janitor.
func InitMiddleware() {
	startCacheJanitor(5 * time.Minute)
}

func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			tokenString = strings.TrimSpace(c.Query("token"))
		}

		if tokenString != "" {
			secret := os.Getenv("JWT_SECRET")
			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return []byte(secret), nil
			})

			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					c.Set("user_id", claims["user_id"])
					c.Set("email", claims["email"])
					c.Set("role", claims["role"])
					c.Set("name", claims["name"])
				}
			}
		}

		c.Next()
	}
}
