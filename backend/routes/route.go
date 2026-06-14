package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/handlers"
	"github.com/khanqais/tradexa/middleware"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/health", func(ctx *gin.Context) {
			ctx.JSON(200, gin.H{
				"status": "Bidding api is working",
			})
		})
		api.POST("/login", handlers.Login)
		api.POST("/register", handlers.Register)
		api.GET("/listings", handlers.GetListings)
		api.GET("/listings/:id", handlers.GetListingByID)

		protected := api.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			protected.POST("/bid", handlers.BidHandler)
			protected.POST("/forget", handlers.ChangePassowrd)
			protected.GET("/me", handlers.GetMe)
			protected.POST("/listings", handlers.CreateListing)
			protected.PUT("/listings/:id", handlers.UpdateListing)
			protected.DELETE("/listings/:id", handlers.DeleteListing)
			protected.POST("/upload", handlers.UploadImage)
			protected.POST("/conversations", handlers.GetOrCreateConversation)
			protected.GET("/conversations", handlers.GetConversationsForUser)
			protected.GET("/conversations/:conversationId/messages", handlers.GetMessagesForConversation)
			protected.GET("/ws/notifications", handlers.NotificationHandler)
			protected.GET("/ws/conversation/:conversationId", handlers.ConversationHandler)

		}
	}

}
