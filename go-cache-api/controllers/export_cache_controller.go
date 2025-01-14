package controllers
import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"go-cache-api/configs"
	"go-cache-api/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	redisClient = configs.ConnectRedis()
	cacheMutex  sync.Mutex
)

const (
	maxAgeDefault = 300

)

func getMaxAgeTime(c echo.Context) int {
	cacheControlHeader := c.Request().Header.Get("Cache-Control")

	if maxAgeIdx := strings.Index(cacheControlHeader, "max-age="); maxAgeIdx != -1 {
		maxAgeStr := cacheControlHeader[maxAgeIdx+len("max-age="):]
		if maxAge, err := strconv.Atoi(strings.Split(maxAgeStr, ",")[0]); err == nil && maxAge > 0 {
			return maxAge
		}
	}

	// fmt.Println(cacheControlHeader)

	return maxAgeDefault
}

func GenerateCacheKey(data string) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	hash := hex.EncodeToString(hasher.Sum(nil))

	eTag := fmt.Sprintf("\"%s\"", hash)

	return eTag
}

func generateETag(data string) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	hash := hex.EncodeToString(hasher.Sum(nil))

	eTag := fmt.Sprintf("\"%s\"", hash)

	return eTag
}

func parseSortFields(sortFields []string) bson.D {
	sorts := bson.D{}
	for _, sort := range sortFields {
		for _, value := range strings.Split(sort, ",") {
			sortOrder := 1
			if strings.HasPrefix(value, "-") {
				sortOrder = -1
				value = strings.TrimPrefix(value, "-")
			}
			sorts = append(sorts, bson.E{Key: value, Value: sortOrder})
		}
	}
	return sorts
}

func generateCacheKey(c echo.Context) string {
	uri := c.Request().Method + ":" + c.QueryParams().Encode()
	maxageString := fmt.Sprintf(":max-age=%s", strconv.Itoa(getMaxAgeTime(c)))

	return "exports:" + uri + maxageString
}

// Etag/if-none-match
func handleIfNoneMatch(c echo.Context, etag string, age int, lastModified time.Time, maxAge int, expire time.Time) error {
	ifNoneMatch := c.Request().Header.Get("If-None-Match")
	if ifNoneMatch != "" && ifNoneMatch == etag {
		setCacheHeaders(c, age, maxAge, etag, expire, lastModified)
		return c.NoContent(http.StatusNotModified)
	}
	return nil
}

// last-modified/if-modified-since
func handleIfModifiedSince(c echo.Context, etag string, age int, lastModified time.Time, maxAge int, expire time.Time) error {
	ifModifiedSince := c.Request().Header.Get("If-Modified-Since")
	if ifModifiedSince != "" && ifModifiedSince == lastModified.UTC().Format(http.TimeFormat) {
		setCacheHeaders(c, age, maxAge, etag, expire, lastModified)
		return c.NoContent(http.StatusNotModified)
	}
	return nil
}

func setCacheHeaders(c echo.Context, age int, maxAge int, etag string, expire time.Time, lastModified time.Time) {
	c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
	c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	c.Response().Header().Set("Etag", etag)
	c.Response().Header().Set("Expires", expire.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("X-Cache-Status", "Hit")
}

// No cache
func handleNoCache(c echo.Context, etag string, lastModified time.Time) error {

	if clientETag := c.Request().Header.Get("If-None-Match"); clientETag != "" {
		if etag == clientETag {
			c.Response().Header().Set("Cache-Control", "no-cache")
			c.Response().Header().Set("Etag", etag)
			c.Response().Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Miss")
			return c.NoContent(http.StatusNotModified)
		}
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Etag", etag)
	c.Response().Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("X-Cache-Status", "Miss")
	return nil
}

func handleCacheOnlyRequest(c echo.Context, ctx context.Context, cacheKey string) error {
	cache, found := redisClient.Get(ctx, cacheKey).Result()

	if found != nil {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("Connection", "close")
		c.Response().Header().Set("X-Cache-Status", "Miss")
		return c.JSON(http.StatusGatewayTimeout, echo.Map{"message": "The resource is not in the cache, and the server could not retrieve it"})
	}

	var exports []models.Product
	if err := json.Unmarshal([]byte(cache), &exports); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error unmarshal JSON"})
	}

	//cache-control: max-age
	maxAge := getMaxAgeTime(c)
	cacheAgeTime, err := redisClient.TTL(ctx, cacheKey).Result()
	if err != nil {
		log.Println(err)
	}

	//ttl of cache in redis
	age := int(maxAge) - int(cacheAgeTime.Seconds())

	//cache time when will expire
	expire := time.Now().Add(time.Duration(maxAge) * time.Second)

	//last modified of resouce
	var lastModified time.Time
	for _, export := range exports {
		if export.UpdatedAt.After(lastModified) {
			lastModified = *export.UpdatedAt
		}
	}

	// etag
	etag := generateETag(cache)

	//if none match
	if err := handleIfNoneMatch(c, etag, age, lastModified, maxAge, expire); err != nil {
		return err
	}

	//if modified since
	if err := handleIfModifiedSince(c, etag, age, lastModified, maxAge, expire); err != nil {
		return err
	}

	setCacheHeaders(c, age, maxAge, etag, expire, lastModified)

	return c.JSON(http.StatusOK, exports)
}

// cache hit
func handleCacheHit(c echo.Context, ctx context.Context, cacheKey string, cachedData string, filter primitive.M , opts *options.FindOptions) error {
	var exports []models.ExportData
	if err := json.Unmarshal([]byte(cachedData), &exports); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error unmarshal JSON"})
	}

	maxAge := getMaxAgeTime(c)
	timeTolive, err := redisClient.TTL(ctx, cacheKey).Result()
	if err != nil {
		log.Println(err)
	}

	age := int(maxAge) - int(timeTolive.Seconds())
	expire := time.Now().Add(time.Duration(maxAge) * time.Second)

	var lastModified time.Time
	for _, product := range exports {
		if product.UpdatedAt.After(lastModified) {
			lastModified = *product.UpdatedAt
		}
	}

	etag := generateETag(cachedData)

	//no cached
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

		etag := generateETag(cachedData)

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

		etag := generateETag(cachedData)

		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("Etag", etag)
		c.Response().Header().Set("Last-Modified", latestUpdateTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("X-Cache-Status", "Miss")

		return c.JSON(http.StatusOK, products)

	}


	if err := handleIfNoneMatch(c, etag, age, lastModified, maxAge, expire); err != nil {
		return err
	}

	if err := handleIfModifiedSince(c, etag, age, lastModified, maxAge, expire); err != nil {
		return err
	}

	setCacheHeaders(c, age, maxAge, etag, expire, lastModified)

	return c.JSON(http.StatusOK, exports)
}

// cache miss
func handleCacheMiss(c echo.Context, ctx context.Context, filter bson.M, opts *options.FindOptions, cacheKey string) error {
	results, err := exportCollection.Find(ctx, filter, opts)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": "Can not find product in collection"})
	}
	defer results.Close(ctx)

	var exports []models.ExportData
	if err := results.All(ctx, &exports); err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"message": err})
	}

	exportsMarshal, err := json.Marshal(exports)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error marshaling JSON"})
	}

	maxAge := getMaxAgeTime(c)

	etag := generateETag(string(exportsMarshal))

	expire := time.Now().Add(time.Duration(maxAge) * time.Second)

	var lastModified time.Time
	for _, export := range exports {
		if export.UpdatedAt.After(lastModified) {
			lastModified = *export.UpdatedAt
		}
	}

	cacheControl := c.Request().Header.Get("Cache-Control")

	if cacheControl == "no-cache" {
		if err := handleNoCache(c, etag, lastModified); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, exports)
	}

	if cacheControl == "no-store" {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("X-Cache-Status", "Miss")
		return c.JSON(http.StatusOK, exports)
	}

	if cacheControl != "no-store" {
		err = redisClient.Set(ctx, cacheKey, exportsMarshal, time.Duration(maxAge)*time.Second).Err()
		if err != nil {
			log.Println(err)
		}
	}

	c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	c.Response().Header().Set("Etag", etag)
	c.Response().Header().Set("Expires", expire.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("Last-Modified", lastModified.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("X-Cache-Status", "Miss")

	return c.JSON(http.StatusOK, exports)
}

// get exports
func ExportsCache(c echo.Context) error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

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
	sorts := parseSortFields(sortFields)

	// Construct filter
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

	cacheKey := generateCacheKey(c)
	if c.Request().Header.Get("Cache-Control") == "only-if-cached" {
		return handleCacheOnlyRequest(c, ctx, cacheKey)
	}

	//find cache in redis
	cachedData, found := redisClient.Get(ctx, cacheKey).Result()

	//cache hit
	if found == nil {
		return handleCacheHit(c, ctx, cacheKey, cachedData, filter, opts )
	}

	//cache miss
	return handleCacheMiss(c, ctx, filter, opts, cacheKey)
}