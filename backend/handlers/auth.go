package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/idtoken"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/middleware"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"omitempty,oneof=buyer seller admin"`
	Otp      string `json:"otp" binding:"required"`
}

type SendOTPInput struct {
	Email string `json:"email" binding:"required,email"`
}

// So this is the structure of the login and register and json means what type of variable we are accepting if not specified
//then We have to send "Email" and binding set the validation rule like min ,max required , omitempty

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type ChangePassowrdInput struct {
	NewPassword string `json:"newpassowrd" binding:"required,min=6"`
	OldPassWord string `json:"oldpassowrd" binding:"required,min=6"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}
	var existing models.User
	normalizedEmail := strings.TrimSpace(strings.ToLower(input.Email))
	err := config.DB.Where("LOWER(email) = ?", normalizedEmail).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "email is already used",
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Verify OTP from Redis
	otpKey := "otp:" + normalizedEmail
	storedOTP, otpErr := config.RDB.Get(context.Background(), otpKey).Result()
	if otpErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP not found or expired. Please request a new OTP."})
		return
	}
	if storedOTP != input.Otp {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid OTP"})
		return
	}

	hashpassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	role := models.RoleBuyer
	if input.Role != "" {
		role = models.Role(input.Role)
	}
	user := models.User{
		Name:     input.Name,
		Email:    normalizedEmail,
		Password: string(hashpassword),
		Role:     role,
	}

	// Create user in DB
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// Delete OTP from Redis — it has been consumed
	config.RDB.Del(context.Background(), otpKey)

	user.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func SendOTP(c *gin.Context) {
	var input SendOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	normalizedEmail := strings.TrimSpace(strings.ToLower(input.Email))

	// Check if user already exists
	var existing models.User
	err := config.DB.Where("LOWER(email) = ?", normalizedEmail).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "email is already registered"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Generate 6-digit OTP
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OTP"})
		return
	}
	otpCode := ""
	for i := 0; i < 6; i++ {
		otpCode += fmt.Sprintf("%d", b[i]%10)
	}

	// Store OTP in Redis with 10-minute TTL (replaces DB write)
	otpKey := "otp:" + normalizedEmail
	if err := config.RDB.Set(context.Background(), otpKey, otpCode, 10*time.Minute).Err(); err != nil {
		log.Printf("failed to store OTP in Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OTP, please try again"})
		return
	}

	// Send OTP email
	subject := "Verify your email - Tradexa"
	htmlBody := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; max-width: 600px; margin: 0 auto; border: 1px solid #e0e0e0; border-radius: 5px;">
			<h2 style="color: #333;">Welcome to Tradexa!</h2>
			<p>Please use the following One-Time Password (OTP) to verify your email address and complete your registration:</p>
			<div style="font-size: 24px; font-weight: bold; background-color: #f7f7f7; padding: 15px; border-radius: 4px; text-align: center; margin: 20px 0; letter-spacing: 5px; color: #4F46E5;">
				%s
			</div>
			<p style="color: #666; font-size: 14px;">This OTP is valid for 10 minutes. If you did not request this, you can safely ignore this email.</p>
			<hr style="border: 0; border-top: 1px solid #eee; margin: 20px 0;">
			<p style="color: #999; font-size: 12px; text-align: center;">Tradexa &copy; 2026. All rights reserved.</p>
		</div>
	`, otpCode)

	if err := utils.SendEmail(normalizedEmail, subject, htmlBody); err != nil {
		log.Printf("failed to send OTP email: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send verification email: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP sent successfully"})
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}
	var user models.User
	normalizedEmail := strings.TrimSpace(strings.ToLower(input.Email))
	if err := config.DB.Where("LOWER(email) = ?", normalizedEmail).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT_SECRET not configured"})
		return
	}
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"name":    user.Name,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("failed to sign jwt token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "failed to generate token",
			"detail": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.Role,
			"picture": user.Avatar,
		},
	})

}
func UploadAvatar(c *gin.Context) {
	userID, _ := c.Get("user_id")

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only jpg, jpeg, png, webp allowed"})
		return
	}
	if header.Size > 3*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file must be under 3MB"})
		return
	}

	uploadResult, err := config.Cloudinary.Upload.Upload(
		context.Background(),
		file,
		uploader.UploadParams{
			Folder:         "tradexa/avatars",
			Transformation: "c_fill,w_200,h_200,q_auto,f_auto",
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload avatar"})
		return
	}

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	user.Avatar = uploadResult.SecureURL
	config.DB.Save(&user)

	c.JSON(http.StatusOK, gin.H{
		"picture": user.Avatar,
	})
}
func ChangePassowrd(c *gin.Context) {
	var input ChangePassowrdInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	userID, _ := c.Get("user_id")
	var user models.User
	config.DB.First(&user, userID)

	err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(input.OldPassWord),
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "incorrect passowrd",
		})
		return
	}
	hashed, _ := bcrypt.GenerateFromPassword(
		[]byte(input.NewPassword),
		bcrypt.DefaultCost,
	)
	user.Password = string(hashed)
	config.DB.Save(&user)
	c.JSON(http.StatusOK, gin.H{
		"message": "password change",
	})
}

func GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")
	email, _ := c.Get("email")
	role, _ := c.Get("role")
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"email":   email,
		"role":    role,
	})
}

type GoogleLoginInput struct {
	Token string `json:"token" binding:"required"`
}

func GoogleLogin(c *gin.Context) {
	var input GoogleLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	payload, err := idtoken.Validate(context.Background(), input.Token, clientID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid google token"})
		return
	}

	email, ok := payload.Claims["email"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email not provided by google"})
		return
	}

	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	var user models.User
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))

	err = config.DB.Where("LOWER(email) = ?", normalizedEmail).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create user
			b := make([]byte, 16)
			rand.Read(b)
			randomPassword := hex.EncodeToString(b)
			hashpassword, _ := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)

			user = models.User{
				Name:     name,
				Email:    normalizedEmail,
				Password: string(hashpassword),
				Role:     models.RoleBuyer,
			}
			if err := config.DB.Create(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT_SECRET not configured"})
		return
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    string(user.Role),
		"name":    user.Name,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.Role,
			"picture": picture,
		},
	})
}

// Logout blacklists the current JWT token in Redis so it cannot be reused.
func Logout(c *gin.Context) {
	rawToken, exists := c.Get("raw_token")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no token found in context"})
		return
	}
	tokenString, ok := rawToken.(string)
	if !ok || tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid token"})
		return
	}

	// Parse to get remaining TTL so the blacklist key auto-expires
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	ttl := 24 * time.Hour // default fallback
	if err == nil && token.Valid {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if exp, ok := claims["exp"].(float64); ok {
				remaining := time.Until(time.Unix(int64(exp), 0))
				if remaining > 0 {
					ttl = remaining
				}
			}
		}
	}

	if err := middleware.BlacklistToken(tokenString, ttl); err != nil {
		log.Printf("[Logout] Failed to blacklist token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed, please try again"})
		return
	}

	log.Printf("[Logout] Token blacklisted for %.0f minutes", ttl.Minutes())
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
