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
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired())
		{
			protected.GET("/me", handlers.GetMe)
		}
	}

}
