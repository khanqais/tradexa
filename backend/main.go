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
	config.DB.AutoMigrate(&models.User{})
	r := gin.Default()
	routes.RegisterRoutes(r)
	r.Run(":8080")
}
