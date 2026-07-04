package main

import (
	"fmt"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/khanqais/tradexa/config"
	"github.com/khanqais/tradexa/handlers"
	"github.com/khanqais/tradexa/middleware"
	"github.com/khanqais/tradexa/models"
	"github.com/khanqais/tradexa/routes"
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
	config.DB.AutoMigrate(
		&models.User{}, &models.OTP{}, &models.Listing{}, &models.ListingImage{},
		&models.Message{}, &models.Conversation{}, &models.Bid{},
		&models.Order{}, &models.ProxyBid{},
	)
	config.RunMigrations(config.DB)

	config.InitQStash()

	config.SetAuctionCloseHandler(workers.TriggerAuctionClose)

	config.SetSSEBroadcaster(handlers.StreamHub.Broadcast)
	if backendURL := os.Getenv("BACKEND_URL"); backendURL != "" {
		fmt.Printf("[QStash] Webhook target: %s/api/internal/auction-close\n", backendURL)
	} else {
		fmt.Println("[QStash] WARNING: BACKEND_URL not set — auction scheduling will use in-process timer fallback")
	}

	middleware.InitMiddleware()

	go workers.StartAuctionSweeper()

	r := gin.Default()

	devOrigins := []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	prodOrigin := os.Getenv("FRONTEND_URL")
	if prodOrigin != "" {
		devOrigins = append(devOrigins, prodOrigin)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:		devOrigins,
		AllowMethods:		[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:		[]string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposeHeaders:		[]string{"Content-Length"},
		AllowCredentials:	true,
		MaxAge:			3600,
	}))

	routes.RegisterRoutes(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
