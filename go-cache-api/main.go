package main

import (
	"go-cache-api/configs"
	"go-cache-api/routes"
	"github.com/labstack/echo"
)

func main() {
	e := echo.New()

	configs.ConnectDB()
	routes.ProductRoute(e)
	routes.ExportRoute(e)
	routes.ExploreRoutes(e)
	routes.UseCaseCache(e)


	// file.InsetProductIntoMongo() //แก้ไฟล์
	// file.InsetExportIntoMongo()	//แก้ไฟล์

	e.Logger.Fatal(e.Start(":8000"))
}
