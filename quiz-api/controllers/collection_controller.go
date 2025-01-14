package controllers

import (
	"context"
	"log"
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
	collectionCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "collections") //connect to collection in database
)

func TimeNow() time.Time {
	localTime, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Fatal(err)
	}
	var timeNow = time.Now().In(localTime)
	return timeNow
}

// ----------------------------------new created function with insert many---------------------------------------------//
func CreateManyCollection(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var collection []models.Collection
	if err := c.Bind(&collection); err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid request payload!"})
	}

	for key, value := range collection {
		if value.Name == "" {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "[" + strconv.Itoa(key) + "]" + "Name is required."})
		}
	}

	updatedAt := TimeNow()
	var newCollections []interface{}
	for _, collection := range collection {
		newCollection := models.Collection{
			ID:        primitive.NewObjectID(),
			Name:      collection.Name,
			CreatedAt: TimeNow(),
			UpdatedAt: &updatedAt,
		}
		newCollections = append(newCollections, newCollection)
	}

	_, err := collectionCollection.InsertMany(ctx, newCollections)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Failed to create new collection."})
	}

	return c.JSON(http.StatusCreated, responses.SuccessResponse{Message: "Collection had been created.", Collection: newCollections})
}


// insert new data into database
func CreateCollection(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var collection models.Collection
	if err := c.Bind(&collection); err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid request payload!"})
	}

	if collection.Name == "" {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Name is required."})
	}

	updateAt := TimeNow()
	newCollection := models.Collection{
		ID:        primitive.NewObjectID(),
		Name:      collection.Name,
		CreatedAt: TimeNow(),
		UpdatedAt: &updateAt,
	}

	_, err := collectionCollection.InsertOne(ctx, newCollection)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "Failed to create new collection!"})
	}

	return c.JSON(http.StatusCreated, responses.SuccessResponse{Message: "Collection had been created.", Collection: newCollection})
}

// get all the data in collections
func GetAllCollections(c echo.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//check limit queryparam
	limit := 10
	if c.QueryParam("limit") != "" {
		limit, err = strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid type limit!"})
		}
	}

	// check page queryparam
	page := 0
	if c.QueryParam("page") != "" {
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid type page!"})
		}
	}

	// page caculate with limit to get offset results
	offset := 0
	if page > 0 {
		offset = (page - 1) * limit
	}

	// check if sorts queryparam
	// แก้ไข การแบ่ง การ sort
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

	filter := bson.M{"deleted_at": bson.M{"$exists": false}}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))

	search := c.QueryParam("search")
	if search != "" {

		filter["name"] = bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}
	}

	if len(sorts) > 0 {
		opts.SetSort(sorts)
	}

	results, err := collectionCollection.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Can not find data in collection."})
	}

	var collections []models.Collection
	if err := results.All(ctx, &collections); err != nil {
		return c.JSON(http.StatusInternalServerError, responses.ErrorResponse{Message: "A Server error occurred."})
	}

	var checkDeletedCollections []models.Collection

	for _, collection := range collections {
		collection.CreatedAt = collection.CreatedAt.Add(7 * time.Hour)

		if collection.UpdatedAt != nil {
			updatedTime := collection.UpdatedAt.Add(7 * time.Hour)
			collection.UpdatedAt = &updatedTime
		} else {
			collection.UpdatedAt = nil
		}
		if collection.DeletedAt == nil {
			checkDeletedCollections = append(checkDeletedCollections, collection)
		}
	}

	if len(checkDeletedCollections) == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No data in collections.", "collections": checkDeletedCollections})
	}

	return c.JSON(http.StatusOK, responses.SuccessResponse{Message: "Get all the collection data.", Collection: checkDeletedCollections})
}

// get collection by id
func GetCollection(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id."})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId, "deleted_at": bson.M{"$exists": false}}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found."})
	}

	collection.CreatedAt = collection.CreatedAt.Add(7 * time.Hour)

	if collection.UpdatedAt != nil {
		updatedAt := collection.UpdatedAt.Add(7 * time.Hour)
		collection.UpdatedAt = &updatedAt
	} else {
		collection.UpdatedAt = nil
	}

	return c.JSON(http.StatusOK, collection)	
}

// update collection by id
func UpdateCollection(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	if err := c.Bind(&collection); err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid request payload"})
	}

	var updateCollection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId}).Decode(&updateCollection)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Collection not found."})
	}

	if collection.Name != "" {
		updateCollection.Name = collection.Name
	}

	updateTime := TimeNow()
	updateCollection.UpdatedAt = &updateTime

	result, err := collectionCollection.UpdateByID(ctx, collectionId, bson.M{"$set": updateCollection})
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Failed to update collection"})
	}

	if result.ModifiedCount == 0 {
		return c.JSON(http.StatusOK, responses.SuccessResponse{Message: "No changes detected."})
	}

	return c.JSON(http.StatusOK, responses.SuccessResponse{Message: "Name had been updated"})
}

// Deleted the collection by its ID.
func DeleteCollection(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Collection not found."})
	}

	result, err := collectionCollection.DeleteOne(ctx, bson.M{"_id": collectionId})
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Failed to delete collection"})
	}

	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, responses.ErrorResponse{Message: "Collection not found"})
	}

	return c.JSON(http.StatusOK, responses.SuccessResponse{Message: collection.Name + " had been deleted"})
}


//------------------------------------new deleted function with condition deleted type-------------------------------------------//

func DeleteCollectionV2(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deleteType, err := strconv.Atoi(c.QueryParam("deleteType"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid delete type"})
	}

	collectionId, err := primitive.ObjectIDFromHex(c.Param("collectionId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid collection id"})
	}

	var collection models.Collection
	err = collectionCollection.FindOne(ctx, bson.M{"_id": collectionId}).Decode(&collection)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Collection not found."})
	}

	var updateCollection bson.M
	if deleteType == 0 {
		_, err := collectionCollection.DeleteOne(ctx, bson.M{"_id": collectionId})
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Failed to hard delete collection"})
		}
	} else if deleteType == 1 {
		updateCollection = bson.M{
			"deleted_at": TimeNow(),
		}

		result, err := collectionCollection.UpdateOne(ctx, bson.M{"_id": collectionId}, bson.M{"$set": updateCollection})
		if err != nil {
			return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Failed to soft delete collection"})
		}

		if result.ModifiedCount == 0 {
			return c.JSON(http.StatusOK, responses.SuccessResponse{Message: "Collection had been deleted"})
		}
	} else {
		return c.JSON(http.StatusBadRequest, responses.ErrorResponse{Message: "Invalid delete type"})
	}

	return c.JSON(http.StatusOK, responses.SuccessResponse{Message: collection.Name + " has been deleted"})
}
