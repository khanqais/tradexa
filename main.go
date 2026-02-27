package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env found")
	}
	config.ConnectDB()
	config.DB.AutoMigrate(&models.User{})

}
