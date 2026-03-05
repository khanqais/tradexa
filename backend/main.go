package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/routes"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env found")
	}
	config.ConnectDB()
	config.ConnectCloudinary()
	config.DB.AutoMigrate(&models.User{}, &models.Listing{}, &models.Message{}, &models.Conversation{})
	r := gin.Default()
	routes.RegisterRoutes(r)
	r.Run(":8080")
}
