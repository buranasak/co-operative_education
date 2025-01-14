package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"go-cache-api/configs"
	"go-cache-api/models"

	"net/http"

	"time"

	"github.com/labstack/echo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	productCollection *mongo.Collection = configs.GetCollection(configs.ConnectDB(), "products")
)

func CreateProducts(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var products []models.Product
	if err := c.Bind(&products); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid request payload", "error": err.Error()})
	}

	for _, value := range products {
		if value.ProductName == "" {
			return c.JSON(http.StatusBadRequest, "Product name is required")
		}
		if value.ValueTHB < 0.0 {
			return c.JSON(http.StatusBadRequest, "Product Value in bath is required")
		}
		if value.ValueUSD < 0.0 {
			return c.JSON(http.StatusBadRequest, "Product Value in dollars is required")
		}
		if value.BusinessSize == "" {
			return c.JSON(http.StatusBadRequest, "Product business size is required")
		}
	}

	timeNow := time.Now()

	var newProducts []interface{}
	for _, product := range products {
		newProduct := models.Product{
			ID:           primitive.NewObjectID(),
			ProductName:  product.ProductName,
			Category:     product.Category,
			ValueTHB:     product.ValueTHB,
			ValueUSD:     product.ValueUSD,
			BusinessSize: product.BusinessSize,
			CreatedAt:    &timeNow,
			UpdatedAt:    &timeNow,
		}
		newProducts = append(newProducts, newProduct)
	}

	_, err := productCollection.InsertMany(ctx, newProducts)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to create product"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Product had been created", "products": newProducts})
}

func GetProducts(c echo.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//check limit queryparam
	limit := 10
	if c.QueryParam("limit") != "" {
		limit, err = strconv.Atoi(c.QueryParam("limit"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid type limit!"})
		}
	}

	// check page queryparam
	page := 0
	if c.QueryParam("page") != "" {
		page, err = strconv.Atoi(c.QueryParam("page"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid type page!"})
		}
	}

	// page caculate with limit to get offset results
	offset := 0
	if page > 0 {
		offset = (page - 1) * limit
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
		}
	}

	if len(sorts) > 0 {
		opts.SetSort(sorts)
	}

	results, err := productCollection.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Can not find data in collection"})
	}

	var products []models.Product
	if err := results.All(ctx, &products); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err})
	}

	//check deleted exist
	var checkProductExist []models.Product
	for _, product := range products {
		product.CreatedAt = product.UpdatedAt

		if product.UpdatedAt != nil {
			updateAt := product.UpdatedAt
			product.UpdatedAt = updateAt
		} else {
			product.UpdatedAt = nil
		}
		// deletedAt part
		checkProductExist = append(checkProductExist, product)
	}

	if len(checkProductExist) == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No data in products", "products": products})
	}

	// count, err := productCollection.CountDocuments(ctx, bson.D{})
	// if err != nil {
	// 	panic(err)
	// }

	return c.JSON(http.StatusOK, echo.Map{"message": "Get all the product data.", "products": products})
}

func GetProduct(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productId, err := primitive.ObjectIDFromHex(c.Param("productId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid product id"})
	}

	var product models.Product
	err = productCollection.FindOne(ctx, bson.M{"_id": productId, "deleted_at": bson.M{"$exists": false}}).Decode(&product)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Product not found"})
	}

	if product.UpdatedAt != nil {
		updateAt := product.UpdatedAt
		product.UpdatedAt = updateAt
	} else {
		product.UpdatedAt = nil
	}

	return c.JSON(http.StatusOK, product)
}

func EditProduct(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productId, err := primitive.ObjectIDFromHex(c.Param("productId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid country id"})
	}

	var product models.Product
	if err := c.Bind(&product); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid request payload"})
	}

	var updateProduct models.Product
	err = productCollection.FindOne(ctx, bson.M{"_id": productId}).Decode(&updateProduct)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Product not found"})
	}

	if product.ProductName != "" {
		updateProduct.ProductName = product.ProductName
	}
	if product.Category != "" {
		updateProduct.Category = product.Category
	}
	if product.ValueTHB != 0 {
		updateProduct.ValueTHB = product.ValueTHB
	}
	if product.ValueUSD != 0 {
		updateProduct.ValueUSD = product.ValueUSD
	}
	if product.BusinessSize != "" {
		updateProduct.BusinessSize = product.BusinessSize
	}

	updateTime := time.Now()
	updateProduct.UpdatedAt = &updateTime

	result, err := productCollection.UpdateByID(ctx, productId, bson.M{"$set": updateProduct})
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to update product"})
	}

	if result.ModifiedCount == 0 {
		return c.JSON(http.StatusOK, echo.Map{"message": "No changes detected"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Product had been updated"})
	
}
func DeleteProduct(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deleteType, err := strconv.Atoi(c.QueryParam("deleteType"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid delete type"})
	}

	productId, err := primitive.ObjectIDFromHex(c.Param("productId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid product id"})
	}

	var product models.Product
	err = productCollection.FindOne(ctx, bson.M{"_id": productId}).Decode(&product)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Product not found."})
	}

	var updateProduct bson.M
	if deleteType == 0 {
		_, err := productCollection.DeleteOne(ctx, bson.M{"_id": productId})
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to hard delete product"})
		}
	} else if deleteType == 1 {
		updateProduct = bson.M{
			"deletedAt": time.Now(),
		}

		result, err := productCollection.UpdateOne(ctx, bson.M{"_id": productId}, bson.M{"$set": updateProduct})
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Failed to soft delete product"})
		}

		if result.ModifiedCount == 0 {
			return c.JSON(http.StatusOK, echo.Map{"message": "product had been deleted"})
		}
	} else {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid delete type"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": product.ProductName + " has been deleted"})
}

// ทดลอง 1 get products
func GetProductsCache(c echo.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cacheControlCheck := c.Request().Header.Get("Cache-Control")

	if !(cacheControlCheck == "no-cache" || cacheControlCheck == "no-store" || cacheControlCheck == "only-if-cached" || strings.Contains(cacheControlCheck, "max-age="+strconv.Itoa(getMaxAgeTime(c))) || cacheControlCheck == "" || cacheControlCheck == "max-age=0") {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Invalid cache-control header request"})
	}

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
			{"product_name": bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}},
			{"category": bson.M{"$regex": primitive.Regex{Pattern: search, Options: "i"}}},
		}
	}


	if len(sorts) > 0 {
		opts.SetSort(sorts)
	}

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	uri := c.Request().Method + c.QueryParams().Encode() + ":max-age=" + strconv.Itoa(getMaxAgeTime(c))

	// uri := c.Request().Method + c.QueryParams().Encode()
	// hashkey := generateETag(uri)

	cacheKey := "products:" + uri
	ifNoneMatch := c.Request().Header.Get("If-None-Match")
	ifModifiedSince := c.Request().Header.Get("If-Modified-Since")

	var pattern string
	index := strings.Index(cacheKey, ":max-age=")
	if index != -1 {
		pattern = cacheKey[:index]
	} else {
		pattern = cacheKey
	}

	cacheKeyCheck, err := redisClient.Keys(ctx, pattern+"*").Result()
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": err})
	}

	var similarKeys []string

	for _, key := range cacheKeyCheck {
		index := strings.Index(key, ":max-age=")
		var trimmedKey string
		if index != -1 {
			trimmedKey = key[:index]
		} else {
			trimmedKey = key
		}

		if trimmedKey == pattern {
			similarKeys = append(similarKeys, key)
		}
	}

	// if len(similarKeys) > 1 {
	// 	return c.JSON(http.StatusBadRequest, echo.Map{"message": "Keys with the same value found ignoring max-age"})
	// }

	// for _, key := range similarKeys {
	// 	fmt.Println(key)
	// }

	// fmt.Println("test",cacheKeyCheck)
	// เอา max-age จาก params ที่รับมา

	cacheMaxAge := strings.Split(cacheKey, ":")
	for _, params := range cacheMaxAge {
		if strings.HasPrefix(params, "max-age=") {
			trimMaxAge := strings.TrimPrefix(params, "max-age=")

			_, err := strconv.Atoi(trimMaxAge)
			if err != nil {
				return c.JSON(http.StatusBadRequest, echo.Map{"message": err})
			}
		}
	}

	// trimMaxAge := strings.TrimPrefix(cacheKey, "max-age")

	// cacheKeyTest, err := redisClient.Keys(ctx , "")

	// cacheKeys, err := redisClient.Keys(ctx, cacheKey).Result()
	// if err != nil {
	// 	return c.JSON(http.StatusNotFound, err)
	// }

	// for _, key := range cacheKeys {
	// 	fmt.Println(key)
	// 	parts := strings.Split(key, ":")
	// 	for _, part := range parts {
	// 		if strings.HasPrefix(part, "max-age=") {
	// 			maxAge := strings.TrimPrefix(part, "max-age=")
	// 			maxAgeValue, err := strconv.Atoi(maxAge)
	// 			if err != nil {
	// 				 return c.JSON(http.StatusBadRequest, echo.Map{"message": err})
	// 			}

	// 			fmt.Println(maxAgeValue)
	// 			if maxAgeValue == getMaxAgeTime(c) {
	// 				continue
	// 			}else if maxAgeValue > getMaxAgeTime(c) || maxAgeValue < getMaxAgeTime(c) {
	// 				return c.JSON(http.StatusBadRequest, echo.Map{"message":"This resouce had been already cached"})
	// 			}
	// 		}
	// 	}
	// }

	// fecth from cache only
	if c.Request().Header.Get("Cache-Control") == "only-if-cached" {
		cacheProducts, found := redisClient.Get(ctx, cacheKey).Result()

		if found != nil {
			c.Response().Header().Set("Cache-Control", "no-store")
			c.Response().Header().Set("Connection", "close")
			c.Response().Header().Set("X-Cache-Status", "Miss")
			return c.JSON(http.StatusGatewayTimeout, echo.Map{"message": "The resource is not in the cache, and the server could not retrieve it"})
		}

		var products []models.Product
		if err := json.Unmarshal([]byte(cacheProducts), &products); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error unmarshal JSON"})
		}

		maxAgeTime := getMaxAgeTime(c)
		cacheAgeTime, err := redisClient.TTL(ctx, cacheKey).Result()
		if err != nil {
			log.Println(err)
		}

		age := int(maxAgeTime) - int(cacheAgeTime.Seconds()) + maxAgeDefault
		expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

		var latestUpdateTime time.Time
		for _, product := range products {
			if product.UpdatedAt.After(latestUpdateTime) {
				latestUpdateTime = *product.UpdatedAt
			}
		}

		cacheEtag := generateETag(cacheProducts)

		if ifNoneMatch != "" && ifNoneMatch == cacheEtag {
			c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
			c.Response().Header().Set("Etag", cacheEtag)
			c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Hit")

			return c.NoContent(http.StatusNotModified)
		}

		// fmt.Println(ifModifiedSince)
		if ifModifiedSince != "" && ifModifiedSince == latestUpdateTime.UTC().Format(http.TimeFormat) {
			c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
			c.Response().Header().Set("Etag", cacheEtag)
			c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Hit")

			return c.NoContent(http.StatusNotModified)
		}

		c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
		c.Response().Header().Set("Etag", cacheEtag)
		c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("X-Cache-Status", "Hit")

		return c.JSON(http.StatusOK, products)
	}

	//find cache if exist or not!
	cacheProducts, found := redisClient.Get(ctx, cacheKey).Result()
	//cache Hit
	if found == nil {

		//no-cache directive
		if c.Request().Header.Get("Cache-Control") == "no-cache" {
			results, err := productCollection.Find(ctx, filter, opts)
			if err != nil {
				return c.JSON(http.StatusNotFound, echo.Map{"message": "Can not find product in collection"})
			}

			var products []models.Product
			if err := results.All(ctx, &products); err != nil {
				return c.JSON(http.StatusNotFound, echo.Map{"message": err})
			}

			maxAgeTime := getMaxAgeTime(c)

			expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

			var latestUpdateTime time.Time
			for _, product := range products {
				if product.UpdatedAt.After(latestUpdateTime) {
					latestUpdateTime = *product.UpdatedAt
				}
			}

			productMarshal, err := json.Marshal(products)
			if err != nil {
				return c.JSON(http.StatusBadRequest, echo.Map{"error": err})
			}

			err = redisClient.Set(ctx, cacheKey, productMarshal, time.Duration(maxAgeTime)*time.Second).Err()
			if err != nil {
				log.Println(err)
			}

			etag := generateETag(cacheProducts)

			c.Response().Header().Set("Cache-Control", "no-cache")
			c.Response().Header().Set("Etag", etag)
			c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Miss")

			return c.JSON(http.StatusOK, products)
		}

		//no-store directive
		if c.Request().Header.Get("Cache-control") == "no-store" {
			results, err := productCollection.Find(ctx, filter, opts)
			if err != nil {
				return c.JSON(http.StatusNotFound, echo.Map{"message": "Can not find product in collection"})
			}

			var products []models.Product
			if err := results.All(ctx, &products); err != nil {
				return c.JSON(http.StatusNotFound, echo.Map{"message": err})
			}

			// maxAgeTime := getMaxAgeTime(c)

			var latestUpdateTime time.Time
			for _, product := range products {
				if product.UpdatedAt.After(latestUpdateTime) {
					latestUpdateTime = *product.UpdatedAt
				}
			}

			// productMarshal, err := json.Marshal(products)
			// if err != nil {
			// 	return c.JSON(http.StatusBadRequest, echo.Map{"error": err})
			// }

			// err = redisClient.Set(ctx, cacheKey, productMarshal, time.Duration(maxAgeTime)*time.Second).Err()

			// if err != nil {
			// 	log.Println(err)
			// }
			etag := generateETag(cacheProducts)

			c.Response().Header().Set("Cache-Control", "no-store")
			c.Response().Header().Set("Etag", etag)
			c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Miss")

			return c.JSON(http.StatusOK, products)

		}

		var products []models.Product
		err = json.Unmarshal([]byte(cacheProducts), &products)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error unmarshal JSON"})
		}

		maxAgeTime := getMaxAgeTime(c)
		cacheAgeTime, err := redisClient.TTL(ctx, cacheKey).Result()
		if err != nil {
			log.Println(err)
		}

		age := int(maxAgeTime) - int(cacheAgeTime.Seconds())
		expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

		var latestUpdateTime time.Time
		for _, product := range products {
			if product.UpdatedAt.After(latestUpdateTime) {
				latestUpdateTime = *product.UpdatedAt
			}
		}

		cacheEtag := generateETag(cacheProducts)

		if ifNoneMatch != "" && ifNoneMatch == cacheEtag {
			c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
			c.Response().Header().Set("Etag", cacheEtag)
			c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Hit")

			return c.NoContent(http.StatusNotModified)
		}

		//either

		if ifModifiedSince != "" && ifModifiedSince == latestUpdateTime.UTC().Format(http.TimeFormat) {
			c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
			c.Response().Header().Set("Etag", cacheEtag)
			c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Hit")

			return c.NoContent(http.StatusNotModified)
		}

		c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
		c.Response().Header().Set("Etag", cacheEtag)
		c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("X-Cache-Status", "Hit")

		return c.JSON(http.StatusOK, products)

		// return c.JSON(http.StatusOK, products)

	} else if err != nil {
		log.Println(err)
	}

	// cache Miss
	results, err := productCollection.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Can not find product in collection"})
	}

	var products []models.Product
	if err := results.All(ctx, &products); err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": err})
	}

	var latestUpdateTime time.Time
	for _, product := range products {
		if product.UpdatedAt.After(latestUpdateTime) {
			latestUpdateTime = *product.UpdatedAt
		}
	}

	productMarshal, err := json.Marshal(products)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error marshaling JSON"})
	}

	maxAgeTime := getMaxAgeTime(c)
	etag := generateETag(string(productMarshal))
	expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

	expiresTimes := expiresTime

	cacheControl := c.Request().Header.Get("Cache-Control")

	//แก้ไข เพิ่มเติม 1
	// If client requested no-cache, return a fresh response without caching
	if cacheControl == "no-cache" {
		// If not modified, return the current ETag and Last-Modified time
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Etag", etag)
		c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("X-Cache-Status", "Miss")
		return c.JSON(http.StatusOK, products)
	}

	// !no-store then cache the resource
	if cacheControl != "no-store" && cacheControl != "no-cache" {
		err = redisClient.Set(ctx, cacheKey, productMarshal, time.Duration(maxAgeTime)*time.Second).Err()
		if err != nil {
			log.Println(err)
		}
	}

	//แก้ไข เพิ่มเติม 2
	if cacheControl == "no-store" {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("X-Cache-Status", "Miss")
		return c.JSON(http.StatusOK, products)
	}

	c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
	c.Response().Header().Set("Etag", etag)
	c.Response().Header().Set("Expires", expiresTimes.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("X-Cache-Status", "Miss")

	return c.JSON(http.StatusOK, products)
}
