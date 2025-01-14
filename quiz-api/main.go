package main

import (
	"net/http"
	"quiz-api/configs"
	"quiz-api/routes"

	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	configs.ConnectDB()

	routes.CollectionRoute(e)
	routes.FeatureRoute(e)

	e.GET("/", func(c echo.Context) error {
		response := map[string]interface{}{
			"message": "quiz-api!",
			"status":  http.StatusOK,
		}
		return c.JSON(http.StatusOK, response)
	})

	e.Logger.Fatal(e.Start(":8000"))
}
