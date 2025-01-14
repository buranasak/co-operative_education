package routes

import (
	"go-cache-api/controllers"

	"github.com/labstack/echo"
)



func ExportRoute(e *echo.Echo){

	//-----------CRUD------------//
	e.POST("/exports", controllers.CreateExports)
	e.GET("/exports", controllers.GetExports)
	e.GET("/exports/:exportId", controllers.GetExport)
	e.PUT("/exports/:exportId", controllers.EditExport)
	e.DELETE("/exports/:exportId", controllers.DeleteExport)


	//------------CACHE--------------// 
	e.GET("/api/v2/exports", controllers.ExportsCache)
}