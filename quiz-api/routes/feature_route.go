package routes

import (
	"quiz-api/controllers"

	"github.com/labstack/echo/v4"
)

func FeatureRoute(e *echo.Echo) {
	e.POST("/collections/:collectionId/items", controllers.CreateFeature) 
	e.GET("/collections/:collectionId/items", controllers.GetAllFeatures) 
	e.GET("/collections/:collectionId/items/:featureId", controllers.GetFeature) 
	e.PUT("/collections/:collectionId/items/:featureId", controllers.UpdateFeature) 
	e.DELETE("/collections/:collectionId/items/:featureId", controllers.DeleteFeature)	

	//-------------------------------------------------------------//
	e.POST("/api/v2/collections/:collectionId/items", controllers.CreateFeatureV2)
	e.DELETE("/api/v2/collections/:collectionId/items/:featureId", controllers.DeletedFeatureV2)
}
