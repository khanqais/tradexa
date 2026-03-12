package middleware

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// first check Authorization header (REST routes)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		// fallback — check ?token= query param (WebSocket routes)
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
			log.Printf("[DEBUG] Auth successful - user_id: %v, email: %v\n", claims["user_id"], claims["email"])
		}

		c.Next()
	}
}
