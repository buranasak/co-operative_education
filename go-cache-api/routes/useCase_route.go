package routes

import (
	// "go-cache-api/controllers"

	"go-cache-api/controllers"

	"github.com/labstack/echo"
)


func UseCaseCache(e *echo.Echo){
	
	e.GET("/private", controllers.PrivateCacheUseCase)

}