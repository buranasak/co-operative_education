package controllers

import (
	"context"
	"go-cache-api/configs"
	"go-cache-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	exportCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "exports")
)

func CreateExports(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var exports []models.ExportData
	if err := c.Bind(&exports); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	for _, export := range exports {
		if export.ProductName == "" {
			return c.JSON(http.StatusBadRequest, "Product name id is required")
		}
		if export.Category == "" {
			return c.JSON(http.StatusBadRequest, "Product category id is required")
		}
		if export.ValueTHB == 0 {
			return c.JSON(http.StatusBadRequest, "Value in baht id is required")
		}
		if export.ValueUSD == 0 {
			return c.JSON(http.StatusBadRequest, "Value in dollars id is required")
		}
		if export.BusinessSize == "" {
			return c.JSON(http.StatusBadRequest, "Business size id is required")
		}
		if export.Country == "" {
			return c.JSON(http.StatusBadRequest, "Country is required")
		}
		if export.Month < 1 {
			return c.JSON(http.StatusBadRequest, "Month is required")
		}
		if export.Year < 1 {
			return c.JSON(http.StatusBadRequest, "Year is required")
		}
	}

	timeNow := time.Now()

	var newExports []interface{}
	for _, export := range exports {
		newExport := models.ExportData{
			ID:        primitive.NewObjectID(),
			ProductName: export.ProductName,
			Category: export.Category,
			ValueTHB: export.ValueTHB,
			ValueUSD:  export.ValueUSD,
			BusinessSize: export.BusinessSize,
			Country:   export.Country,
			Month:     export.Month,
			Year:      export.Year,
			CreatedAt: &timeNow,
			UpdatedAt: &timeNow,
		}

		newExports = append(newExports, newExport)
	}

	_, err := exportCollection.InsertMany(ctx, newExports)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Failed to create export"})
	}
	return c.JSON(http.StatusCreated, echo.Map{"exports": newExports})
}

func GetExports(c echo.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	//check limit queryparam
	limit := 10
	if c.QueryParam("limit") != "" {
		limit, err = strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid type limit!"})
		}
	}

	offset := 0
	if c.QueryParam("offset") != "" {
		offset, err = strconv.Atoi(c.QueryParam("offset"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid type offset!"})
		}
	}

	// check if sorts queryparam
	// แก้ไข การแบ่ง การ sort
	sortFields := c.QueryParams()["sortby"]
	sorts := bson.D{}
	if len(sortFields) > 0 {
		for _, sort := range sortFields {
			for _, value := range strings.Split(sort, ",") {
				sortOrder := 1
				if strings.HasPrefix(value, "-") {
					sortOrder = -1
					value = strings.Trim(value, "-")

				}
				sorts = append(sorts, bson.E{Key: value, Value: sortOrder})
			}
		}
	}

	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))

	search := c.QueryParam("search")
	if search != "" {
		filter["$or"] = []bson.M{
			{"productName": bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}},
			{"category": bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}},
			{"country": bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}},
		}
	}

	if len(sorts) > 0 {
		opts.SetSort(sorts)
	}

	results, err := exportCollection.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Can not find data in exports"})
	}

	var exports []models.ExportData
	if err := results.All(ctx, &exports); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err})
	}

	return c.JSON(http.StatusOK, exports)
}

func GetExport(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exportId, err := primitive.ObjectIDFromHex(c.Param("exportId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid product id"})
	}

	var export models.ExportData
	err = exportCollection.FindOne(ctx, bson.M{"_id": exportId, "deleted_at": bson.M{"$exists": false}}).Decode(&export)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Export not found"})
	}

	exportWithProduct := models.ExportData{
		ID:           export.ID,
		ProductName:  export.ProductName,
		Category:     export.Category,
		ValueTHB:     export.ValueTHB,
		ValueUSD:     export.ValueUSD,
		BusinessSize: export.BusinessSize,
		Country:      export.Country,
		Month:        export.Month,
		Year:         export.Year,
		CreatedAt:    export.CreatedAt,
		UpdatedAt:    export.UpdatedAt,
	}

	return c.JSON(http.StatusOK, exportWithProduct)
}

func EditExport(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exportId, err := primitive.ObjectIDFromHex(c.Param("exportId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid export id"})
	}

	var export models.ExportData
	if err := c.Bind(&export); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid request payload"})
	}

	var updateExport models.ExportData
	err = exportCollection.FindOne(ctx, bson.M{"_id": exportId}).Decode(&updateExport)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Export not found"})
	}

	if export.ProductName != "" {
		updateExport.ProductName = export.ProductName
	}
	if export.Category != "" {
		updateExport.Category = export.Category
	}
	if export.ValueTHB != 0 {
		updateExport.ValueTHB = export.ValueTHB
	}
	if export.ValueUSD != 0 {
		updateExport.ValueUSD = export.ValueUSD
	}
	if export.Country != "" {
		updateExport.Country = export.Country
	}
	if export.Month != 0 {
		updateExport.Month = export.Month
	}
	if export.Year != 0 {
		updateExport.Year = export.Year
	}

	updateTime := time.Now()
	updateExport.UpdatedAt = &updateTime

	result, err := exportCollection.UpdateByID(ctx, exportId, bson.M{"$set": updateExport})
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to update export"})
	}

	if result.ModifiedCount == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No changes detected"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Export had been updated"})
}

func DeleteExport(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deleteType, err := strconv.Atoi(c.QueryParam("deleteType"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid delete type"})
	}

	exportId, err := primitive.ObjectIDFromHex(c.Param("exportId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid export id"})
	}

	var export models.Product
	err = exportCollection.FindOne(ctx, bson.M{"_id": exportId}).Decode(&export)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Export not found"})
	}

	var updateExport bson.M
	if deleteType == 0 {
		_, err := exportCollection.DeleteOne(ctx, bson.M{"_id": exportId})
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to hard delete export"})
		}
	} else if deleteType == 1 {
		updateExport = bson.M{
			"deletedAt": time.Now(),
		}

		result, err := exportCollection.UpdateOne(ctx, bson.M{"_id": exportId}, bson.M{"$set": updateExport})
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to soft delete export"})
		}

		if result.ModifiedCount == 0 {
			return c.JSON(http.StatusOK, echo.Map{"message": "Export had been deleted"})
		}
	} else {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid delete type"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": export.ID.Hex() + " has been deleted"})
}



