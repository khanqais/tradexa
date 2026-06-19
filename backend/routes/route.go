package routes

import (
	"time"

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

		api.POST("/login", middleware.RateLimit("rl:login", middleware.ByIP, 10, 15*time.Minute), handlers.Login)

		api.POST("/register", handlers.Register)

		api.POST("/auth/send-otp", middleware.RateLimit("rl:otp", middleware.ByIP, 5, 1*time.Hour), handlers.SendOTP)
		api.POST("/auth/forgot-password/send-otp", middleware.RateLimit("rl:forgot", middleware.ByIP, 3, 1*time.Hour), handlers.ForgotPasswordSendOTP)
		api.POST("/auth/forgot-password/reset", handlers.ForgotPasswordReset)

		api.POST("/auth/google", handlers.GoogleLogin)
		api.GET("/listings", handlers.GetListings)
		api.GET("/listings/:id", middleware.ValidateParamInt("id"), middleware.OptionalAuth(), handlers.GetListingByID)
		api.GET("/stream/:id", middleware.ValidateParamInt("id"), handlers.StreamBid)

		api.POST("/payment/webhook", handlers.WebhookPayment)

		protected := api.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			protected.POST("/logout", handlers.Logout)
			protected.POST("/bid", handlers.BidHandler)
			protected.POST("/forget", handlers.ChangePassowrd)
			protected.GET("/me", handlers.GetMe)
			protected.POST("/me/avatar", handlers.UploadAvatar)
			protected.POST("/listings", handlers.CreateListing)
			protected.PUT("/listings/:id", middleware.ValidateParamInt("id"), handlers.UpdateListing)
			protected.DELETE("/listings/:id", middleware.ValidateParamInt("id"), handlers.DeleteListing)
			protected.POST("/upload", handlers.UploadImage)
			protected.POST("/conversations", handlers.GetOrCreateConversation)
			protected.GET("/conversations", handlers.GetConversationsForUser)
			protected.GET("/conversations/:conversationId/messages", middleware.ValidateParamInt("conversationId"), handlers.GetMessagesForConversation)
			protected.POST("/payment/create-order", handlers.CreateCashfreeOrder)
			protected.POST("/payment/verify", handlers.VerifyPayment)
			protected.POST("/orders/:id/ship", middleware.ValidateParamInt("id"), handlers.MarkOrderShipped)
			protected.GET("/ws/notifications", handlers.NotificationHandler)
			protected.GET("/ws/conversation/:conversationId", middleware.ValidateParamInt("conversationId"), handlers.ConversationHandler)
		}
	}

}
