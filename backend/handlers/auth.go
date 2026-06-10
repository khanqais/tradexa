package handlers

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"omitempty,oneof=buyer seller admin"`
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
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"erorr": "failed to create user",
		})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
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
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
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
