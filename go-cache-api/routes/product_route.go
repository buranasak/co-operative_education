package routes

import (
	"go-cache-api/controllers"

	"github.com/labstack/echo"
)

func ProductRoute(e *echo.Echo){
	e.POST("/products", controllers.CreateProducts)
	e.GET("/products", controllers.GetProducts)
	e.GET("/products/:productId", controllers.GetProduct)
	e.PUT("/products/:productId", controllers.EditProduct)
	e.DELETE("/products/:productId", controllers.DeleteProduct)

	e.GET("/api/v2/products", controllers.GetProductsCache)
}