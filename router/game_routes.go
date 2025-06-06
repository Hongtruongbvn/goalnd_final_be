package routes

import (
	controllers "go-mvc-demo/controller"

	"github.com/gin-gonic/gin"
)

func GameRoutes(r *gin.Engine) {
	// protected := r.Group("/api")
	// protected.Use(middleware.AuthMiddleware())
	// {
	// 	protected.GET("/my-purchases", controllers.GetPurchasedGames)
	// 	protected.GET("/my-rentals", controllers.GetRentedGames)
	// }

	game := r.Group("/games")
	{
		r.GET("/fetch-games", controllers.FetchGamesByPage)
		r.GET("/fetch-games100", controllers.FetchAndSaveGames100)

		game.POST("/", controllers.CreateGame)
		game.GET("/", controllers.GetGames)
		game.GET("/:id", controllers.GetGameByID)
		game.DELETE("/:id", controllers.DeleteGame)
		game.GET("/fetch", controllers.FetchAndSaveGames)
	}
}
