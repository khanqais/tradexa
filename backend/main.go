package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/hibiken/asynq"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/routes"
	"github.com/khanqais/tradexa/tasks"
	"github.com/khanqais/tradexa/workers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No .env found")
	}
	config.ConnectDB()
	config.ConnectCloudinary()
	config.ConnectRedis()
	config.DB.AutoMigrate(&models.User{}, &models.OTP{}, &models.Listing{}, &models.ListingImage{}, &models.Message{}, &models.Conversation{}, &models.Bid{}, &models.Order{}, &models.ProxyBid{})
	config.RunMigrations(config.DB)

	config.InitAsynq()
	go startAsynqWorker()
	r := gin.Default()

	devOrigins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	prodOrigin := os.Getenv("FRONTEND_URL")
	if prodOrigin != "" {
		devOrigins = append(devOrigins, prodOrigin)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     devOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           3600,
	}))

	routes.RegisterRoutes(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func startAsynqWorker() {
	mux := asynq.NewServeMux()
	mux.HandleFunc(tasks.TypeAuctionClose, workers.HandleAuctionCloseTask)

	if err := config.AsynqServer.Run(mux); err != nil {
		log.Fatalf("Could not run Asynq server: %v", err)
	}
}
