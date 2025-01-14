package controllers

import (
	"context"
	"net/http"
	"quiz-api/configs"
	"quiz-api/models"
	"quiz-api/responses"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	featureCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "features") //connect to the features collection
)

// insert new feature data
func CreateFeature(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	var feature models.Feature
	if err := c.Bind(&feature); err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid request payload"})
	}

	if feature.Geometry.Type == "" {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Type is required."})
	}

	if feature.Geometry.Coordinates == nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Coordinates is required."})
	}

	if feature.Properties["name"] == nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Name in properties is required."})
	}

	//check coordinate must be numeric
	for _, coord := range feature.Geometry.Coordinates {
		if _, ok := coord.(string); ok {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Coordinate must be numeric"})
		}
	}

	feature.Properties["collectionId"] = collectionId

	newFeature := models.Feature{
		Id:         primitive.NewObjectID(),
		Type:       "Feature",
		Geometry:   feature.Geometry,
		Properties: feature.Properties,
		CreatedAt:  TimeNow(),
	}

	_, err = featureCollection.InsertOne(ctx, newFeature)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to create new feature"})
	}

	return c.JSON(http.StatusCreated, responses.SuccessFeatureResponse{Message: "Created new feature successfully", Feature: newFeature})
}

// get all the feature data by collcection id
func GetAllFeatures(c echo.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	limit := 10
	if c.QueryParam("limit") != "" {
		limit, err = strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid limit type!"})
		}
	}

	page := 0
	if c.QueryParam("page") != "" {
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid page type!"})
		}
	}

	offset := 0
	if page > 0 {
		offset = (page - 1) * limit
	}

	sortFields := c.QueryParams()["sort_by"]

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

	filter := bson.M{"properties.collectionId": collectionId, "deleted_at": bson.M{"$exists": false}}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))

	search := c.QueryParam("search")
	if search != "" {
		filter["properties.name"] = bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
	}

	if len(sorts) > 0 {
		opts.SetSort(sorts)
	}

	results, err := featureCollection.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "A server error occurred while fetching features"})
	}

	var features []models.Feature
	if err := results.All(ctx, &features); err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "A server error occurred"})
	}

	var checkDeletedFeatures []models.Feature
	for _, feature := range features {
		feature.CreatedAt = feature.CreatedAt.Add(7 * time.Hour)

		if feature.UpdatedAt != nil {
			updatedTime := feature.UpdatedAt.Add(7 * time.Hour)
			feature.UpdatedAt = &updatedTime
		} else {
			feature.UpdatedAt = nil
		}
		if collection.DeletedAt == nil {
			checkDeletedFeatures = append(checkDeletedFeatures, feature)
		}
	}

	if len(checkDeletedFeatures) == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No data in feature.", "features": checkDeletedFeatures})
	}

	return c.JSON(http.StatusOK, responses.SuccessFeatureResponse{Message: "Get all the feature data", Feature: checkDeletedFeatures})
}

// get feature by feature id
func GetFeature(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	featureId, err := primitive.ObjectIDFromHex(c.Param("featureId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid feature id"})
	}

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	var features models.Feature
	err = featureCollection.FindOne(ctx, bson.M{"_id": featureId, "deleted_at": bson.M{"$exists": false}, "properties.collectionId": collectionId}).Decode(&features)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Feature collection not found"})
		}
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "A Server error occurred."})
	}

	return c.JSON(http.StatusOK, features)
}

// update feature by feature id
func UpdateFeature(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	featureId, err := primitive.ObjectIDFromHex(c.Param("featureId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid feature id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	var feature models.Feature
	if err := c.Bind(&feature); err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid request payload"})
	}

	var updateFeature models.Feature
	err = featureCollection.FindOne(ctx, bson.M{"_id": featureId}).Decode(&updateFeature)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Feature not found"})
	}

	if feature.Geometry.Type != "" {
		updateFeature.Geometry.Type = feature.Geometry.Type
	}

	if feature.Geometry.Coordinates != nil {
		updateFeature.Geometry.Coordinates = feature.Geometry.Coordinates
	}

	if feature.Properties != nil {
		updateFeature.Properties = feature.Properties
	}

	if feature.Properties != nil && updateFeature.Properties != nil {
		updateFeature.Properties["collectionId"] = collectionId
	}

	updateTime := TimeNow()
	updateFeature.UpdatedAt = &updateTime

	result, err := featureCollection.UpdateByID(ctx, featureId, bson.M{"$set": updateFeature})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to update feature"})
	}

	if result.ModifiedCount == 0 {
		return c.JSON(http.StatusNoContent, responses.SuccessResponse{Message: "No changes detected."})
	}

	return c.JSON(http.StatusOK, responses.SuccessResponse{Message: "Feature updated successfully"})
}

// deleate feature by its id
func DeleteFeature(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	//count collection document in database to check the existence of the collection data
	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	if !collection.DeletedAt.IsZero() {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "This collection had been deleted."})
	}

	objFeatureID, err := primitive.ObjectIDFromHex(c.Param("featureId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid feature id"})
	}

	result, err := featureCollection.DeleteOne(ctx, bson.M{"_id": objFeatureID, "properties.collectionId": collectionId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to delete feature"})
	}
	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Feature collection not found"})
	}

	return c.JSON(http.StatusOK, responses.SuccessResponse{Message: "Feature collection had been deleted"})
}

// ----------------------------------new created function with insert many---------------------------------------------//
func CreateFeatureV2(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	var features []models.Feature
	if err := c.Bind(&features); err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid request payload!"})
	}

	for _, value := range features {
		if value.Geometry.Type == "" {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Geometry type is required."})
		}
		if value.Geometry.Coordinates == nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Geometry coordinates is required."})
		}
		if value.Properties["name"] == nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Name in properties is required."})
		}
	}

	// Check coordinates numeric
	for _, feature := range features {
		for _, coord := range feature.Geometry.Coordinates {
			if _, ok := coord.(float64); !ok {
				return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Coordinate must be numeric"})
			}
		}
		feature.Properties["collectionId"] = collectionId
	}

	var newFeatures []interface{}
	for _, feature := range features {
		newFeature := models.Feature{
			Id:         primitive.NewObjectID(),
			Type:       "Feature",
			Geometry:   feature.Geometry,
			Properties: feature.Properties,
			CreatedAt:  TimeNow(),
		}
		newFeatures = append(newFeatures, newFeature)
	}

	_, err = featureCollection.InsertMany(ctx, newFeatures)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to create new feature"})
	}

	return c.JSON(http.StatusCreated, responses.SuccessFeatureResponse{Message: "Created new features successfully", Feature: newFeatures})
}

// ------------------------------------new deleted function with condition deleted type-------------------------------------------//
func DeletedFeatureV2(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deleteType, err := strconv.Atoi(c.QueryParam("deleteType"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid deletion type"})
	}

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	featureId, err := primitive.ObjectIDFromHex(c.Param("featureId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid feature id"})
	}

	var features models.Feature
	err = featureCollection.FindOne(ctx, bson.M{"_id": featureId}).Decode(&features)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Feature not found."})
	}

	var updateFeature bson.M
	if deleteType == 0 {
		_, err := featureCollection.DeleteOne(ctx, bson.M{"_id": featureId})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to hard delete collection"})
		}
	} else if deleteType == 1 {
		updateFeature = bson.M{"deleted_at": TimeNow()}
		result, err := featureCollection.UpdateOne(ctx, bson.M{"_id": featureId}, bson.M{"$set": updateFeature})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to soft delete feature"})
		}
		if result.ModifiedCount == 0 {
			return c.JSON(http.StatusOK, responses.SuccessFeatureResponse{Message: "Feature had been deleted."})
		}
	} else {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid delete type."})
	}

	return c.JSON(http.StatusOK, responses.SuccessFeatureResponse{Message: "Feature had been deleted"})
}
