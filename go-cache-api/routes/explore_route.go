package routes

import (
	"go-cache-api/configs"
	"go-cache-api/controllers"
	"log"

	"github.com/labstack/echo"
)


func ExploreRoutes(e *echo.Echo) {
	db, err := configs.Connect("mongodb://localhost:27017")
	if err != nil {
		log.Fatalln(err)
	}

	handler := &controllers.Handler{
		DB: db,
	}

	e.POST("/explore", handler.ExploreServiceUsages)
}