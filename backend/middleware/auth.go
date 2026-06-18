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

		log.Printf("[DEBUG] Auth check - Path: %s, Token present: %v\n", c.Request.URL.Path, tokenString != "")

		if tokenString == "" {
			log.Println("[ERROR] authorization token missing")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization token missing"})
			c.Abort()
			return
		}

		blacklistKey := "blacklist:" + tokenString
		blacklisted, _ := config.RDB.Exists(context.Background(), blacklistKey).Result()
		if blacklisted > 0 {
			log.Println("[ERROR] Token is blacklisted (logged out)")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked, please login again"})
			c.Abort()
			return
		}

		secret := os.Getenv("JWT_SECRET")
		log.Printf("[DEBUG] JWT_SECRET length: %d\n", len(secret))

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
			log.Printf("[DEBUG] Token claims: %v\n", claims)
			c.Set("user_id", claims["user_id"])
			c.Set("email", claims["email"])
			c.Set("role", claims["role"])
			c.Set("name", claims["name"])
			c.Set("raw_token", tokenString)
			log.Printf("[DEBUG] Auth successful - user_id: %v, email: %v\n", claims["user_id"], claims["email"])
		}

		c.Next()
	}
}

func BlacklistToken(tokenString string, ttl time.Duration) error {
	ctx := context.Background()
	return config.RDB.Set(ctx, "blacklist:"+tokenString, "1", ttl).Err()
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
