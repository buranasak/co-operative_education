package file

import (
	"context"
	"fmt"
	"go-cache-api/configs"
	"go-cache-api/models"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/tealeg/xlsx"
	// "go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	// productCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "products")
	exportCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "exports")
)

func InsetExportIntoMongo() {
	excelFileName := "file/exportdata.xlsx"

	xlFile, err := xlsx.OpenFile(excelFileName)
	if err != nil {
		log.Fatal("Error opening Excel file:", err)
	}

	sheet := xlFile.Sheets[0]

	var exportdata []models.ExportData

	for rowIndex, row := range sheet.Rows {
		if rowIndex == 0 {
			continue
		}

		if row != nil && len(row.Cells) >= 8 {

			valueTHBStr := strings.TrimSpace(row.Cells[4].String())
			valueUSDStr := strings.TrimSpace(row.Cells[5].String())
			valueTHB, err := strconv.Atoi(valueTHBStr)
			if err != nil {
				fmt.Printf("Error converting ValueTHB for row %d: %s\n", rowIndex, err)
				continue
			}
			valueUSD, err := strconv.Atoi(valueUSDStr)
			if err != nil {
				fmt.Printf("Error converting ValueUSD for row %d: %s\n", rowIndex, err)
				continue
			}

			monthStr := strings.TrimSpace(row.Cells[6].String())
			monthInt, err := strconv.Atoi(monthStr)
			if err != nil {
				fmt.Println(err)
				continue
			}

			yearStr := strings.TrimSpace(row.Cells[7].String())
			yearInt, err := strconv.Atoi(yearStr)
			if err != nil {
				fmt.Println(err)
				continue
			}

			createAt := time.Now()

			data := models.ExportData{
				ID:           primitive.NewObjectID(),
				ProductName:  row.Cells[2].String(),
				Category:     row.Cells[1].String(),
				ValueTHB:     valueTHB,
				ValueUSD:     valueUSD,
				BusinessSize: row.Cells[3].String(),
				Country:      row.Cells[0].String(),
				Month:        monthInt,
				Year:         yearInt,
				CreatedAt:    &createAt,
				UpdatedAt:    &createAt,
			}
			exportdata = append(exportdata, data)
		}
	}

	var exportInterfaces []interface{}
	for _, exportInterface := range exportdata {
		exportInterfaces = append(exportInterfaces, exportInterface)
	}

	_, err = exportCollection.InsertMany(context.TODO(), exportInterfaces)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

// func InsetExportIntoMongo() {
// 	excelFileName := "file/test1.xlsx"

// 	xlFile, err := xlsx.OpenFile(excelFileName)
// 	if err != nil {
// 		log.Fatal("Error opening Excel file:", err)
// 	}

// 	sheet := xlFile.Sheets[1]

// 	var exports []models.Export

// 	for rowIndex, row := range sheet.Rows {
// 		if rowIndex == 0 {
// 			continue
// 		}

// 		if row != nil && len(row.Cells) >= 4 {

// 			productIDStr := strings.TrimSpace(row.Cells[0].String())
// 			productId, err := primitive.ObjectIDFromHex(productIDStr)
// 			if err != nil {
// 				fmt.Println(err)
// 				continue
// 			}
// 			var product models.Product
// 			err = productCollection.FindOne(context.Background(), bson.M{"_id": productId}).Decode(&product)
// 			if err != nil {
// 				fmt.Println(err)
// 			}

// 			monthStr := strings.TrimSpace(row.Cells[2].String())
// 			monthInt, err := strconv.Atoi(monthStr)
// 			if err != nil {
// 				fmt.Println(err)
// 				continue
// 			}

// 			yearStr := strings.TrimSpace(row.Cells[3].String())
// 			yearInt, err := strconv.Atoi(yearStr)
// 			if err != nil {
// 				fmt.Println(err)
// 				continue
// 			}

// 			createAt := time.Now()

// 			data := models.Export{
// 				ID: primitive.NewObjectID(),
// 				Product: models.Product{
// 					ID:           productId,
// 					ProductName:  product.ProductName,
// 					Category:     product.Category,
// 					ValueTHB:     product.ValueTHB,
// 					ValueUSD:     product.ValueUSD,
// 					BusinessSize: product.BusinessSize,
// 					CreatedAt:    product.CreatedAt,
// 					UpdatedAt:    product.UpdatedAt,
// 				},
// 				Country:   row.Cells[1].String(),
// 				Month:     monthInt,
// 				Year:      yearInt,
// 				CreatedAt: &createAt,
// 				UpdatedAt: &createAt,
// 			}
// 			exports = append(exports, data)
// 		}
// 	}

// 	var interfaceSlice []interface{}
// 	for _, productType := range exports {
// 		interfaceSlice = append(interfaceSlice, productType)
// 	}

// 	_, err = exportCollection.InsertMany(context.TODO(), interfaceSlice)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	}
// }

// func InsetExportIntoMongo() {
// 	excelFileName := "file/test1.xlsx"

// 	xlFile, err := xlsx.OpenFile(excelFileName)
// 	if err != nil {
// 		log.Fatal("Error opening Excel file:", err)
// 	}

// 	sheet := xlFile.Sheets[1]

// 	var exports []models.Export

// 	for rowIndex, row := range sheet.Rows {
// 		if rowIndex == 0 {
// 			continue
// 		}

// 		if row != nil && len(row.Cells) >= 4 {

// 			productIDStr := strings.TrimSpace(row.Cells[0].String())
// 			productId, err := primitive.ObjectIDFromHex(productIDStr)
// 			if err != nil {
// 				fmt.Println(err)
// 				continue
// 			}
// 			var product models.Product
// 			err = productCollection.FindOne(context.Background(), bson.M{"_id": productId}).Decode(&product)
// 			if err != nil {
// 				fmt.Println(err)
// 			}

// 			monthStr := strings.TrimSpace(row.Cells[2].String())
// 			monthInt, err := strconv.Atoi(monthStr)
// 			if err != nil {
// 				fmt.Println(err)
// 				continue
// 			}

// 			yearStr := strings.TrimSpace(row.Cells[3].String())
// 			yearInt, err := strconv.Atoi(yearStr)
// 			if err != nil {
// 				fmt.Println(err)
// 				continue
// 			}

// 			createAt := time.Now()

// 			data := models.Export{
// 				ID: primitive.NewObjectID(),
// 				ProductId: productId,
// 				Country:   row.Cells[1].String(),
// 				Month:     monthInt,
// 				Year:      yearInt,
// 				CreatedAt: &createAt,
// 				UpdatedAt: &createAt,
// 			}
// 			exports = append(exports, data)
// 		}
// 	}

// 	var interfaceSlice []interface{}
// 	for _, productType := range exports {
// 		interfaceSlice = append(interfaceSlice, productType)
// 	}

// 	_, err = exportCollection.InsertMany(context.TODO(), interfaceSlice)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	}
// }
