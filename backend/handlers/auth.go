package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"omitempty,oneof=buyer seller admin"`
}

// So this is the structure of the login and register and json means what type of variable we are accepting if not specified
//then We have to send "Email" and binding set the validation rule like min ,max required , omitempty

type Login struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error()})
		return
	}
	var existing models.User
	if err=
}
