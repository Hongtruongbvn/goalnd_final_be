package main

import (
	"go-mvc-demo/config"
	controllers "go-mvc-demo/controller"
	"go-mvc-demo/middleware"
	routes "go-mvc-demo/router"
	"math/rand"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	config.ConnectDB()

	r := gin.Default()
	rand.Seed(time.Now().UnixNano())

	// Configure CORS middleware properly at the beginning
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Specify your frontend origin
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
	}

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", func(c *gin.Context) {
			userID := c.MustGet("user_id").(string)
			c.JSON(200, gin.H{"user_id": userID})
		})
		protected.GET("/my-purchases", controllers.GetPurchasedGames)
		protected.GET("/my-rentals", controllers.GetRentedGames)

	}
	routes.UserRoutes(r)
	routes.GameRoutes(r)
	routes.TransactionRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
