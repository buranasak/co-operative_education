package routes

import (
	"quiz-api/controllers"

	"github.com/labstack/echo/v4"
)

func CollectionRoute(e *echo.Echo) {
	e.POST("/collections", controllers.CreateCollection) 
	e.GET("/collections", controllers.GetAllCollections) 
	e.GET("/collections/:collectionId", controllers.GetCollection) 
	e.PUT("/collections/:collectionId", controllers.UpdateCollection) 
	e.DELETE("/collections/:collectionId", controllers.DeleteCollection) 
	
	//-------------------------------------------------------------//
	e.POST("/api/v2/collections", controllers.CreateManyCollection)
	e.DELETE("/api/v2/collections/:collectionId", controllers.DeleteCollectionV2)
}