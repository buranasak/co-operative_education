package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-cache-api/configs"
	"go-cache-api/models"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"go.mongodb.org/mongo-driver/bson"
)

type Handler struct {
	DB *configs.Database
}

func IntToPointer(i int) *int {
	return &i
}

// IsStringInSlice function
func IsStringInSlice(s string, lists []string) bool {
	for _, v := range lists {
		if s == v {
			return true
		}
	}
	return false
}

func (h *Handler) ExploreServiceUsages(c echo.Context) error {
	var err error

	body := new(models.ExploreRequest)

	err = c.Bind(body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &models.Exception{
			Status: IntToPointer(http.StatusBadRequest),
			Detail: err.(*echo.HTTPError).Message.(string),
		})
	}

	requestBodyJSON, err := json.Marshal(body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	specificResponse := string(requestBodyJSON) + strconv.Itoa(getMaxAgeTime(c))
	hashkey := generateETag(specificResponse)
	cacheKey := "products:" + hashkey

	pipeline := []bson.M{}
	match := bson.M{}

	//
	// filter stage
	//
	if body.Filter != nil {

		arguments := body.Filter.Arguments

		args := []interface{}{}

		for _, a := range arguments {
			b, _ := json.Marshal(a)

			var arg *models.ExploreFilter
			err = json.Unmarshal(b, &arg)
			if err != nil {
				return c.JSON(http.StatusBadRequest, &models.Exception{
					Status: IntToPointer(http.StatusBadRequest),
					Detail: "Body 'filter' is invalid",
				})
			}

			if len(arg.Arguments) > 0 {
				args = append(args, arg)
			}

		}

		body.Filter.Arguments = args

		match, err = configs.FilterToBsonM(body.Filter)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &models.Exception{
				Status: IntToPointer(http.StatusBadRequest),
				Detail: "Body 'filter' is invalid, " + err.Error(),
			})
		}
	}

	//
	// match stage
	//
	if len(match) > 0 {
		pipeline = append(pipeline, bson.M{
			"$match": match,
		})
	}

	//
	// group
	//
	groupId := bson.M{}

	//columns
	for _, col := range body.Columns {
		name := strings.ReplaceAll(col.Name, ".", "_")
		groupId[name] = "$" + configs.ChangeKeyId(col.Name)
	}

	group := bson.M{}

	if len(groupId) > 0 {
		group["_id"] = groupId

	} else {
		group["_id"] = nil
	}

	//aggregate
	for _, ag := range body.Aggregate {
		column := strings.ReplaceAll(ag.Column, ".", "_")

		aggregate := bson.M{}

		if strings.ToLower(ag.Aggregate) == "count" {
			aggregate["$sum"] = 1
		} else {
			aggregate["$"+ag.Aggregate] = "$" + configs.ChangeKeyId(ag.Column)
		}
		group[column] = aggregate
	}

	pipeline = append(pipeline, bson.M{
		"$group": group,
	})

	//
	// project
	//
	project := bson.M{
		"_id": 0,
	}

	for _, col := range body.Columns {
		alias := col.Alias
		if alias == "" {
			alias = col.Name
		}
		project[alias] = "$_id." + strings.ReplaceAll(col.Name, ".", "_")
	}

	for _, ag := range body.Aggregate {
		column := strings.ReplaceAll(ag.Column, ".", "_")

		project[ag.Alias] = "$" + column
	}

	pipeline = append(pipeline, bson.M{
		"$project": project,
	})

	//
	// sort
	//
	sort := bson.D{}

	for _, col := range body.Columns {
		alias := col.Alias
		if alias == "" {
			alias = col.Name
		}

		sort = append(sort, bson.E{Key: alias, Value: 1})
	}

	if len(body.Sorts) > 0 {
		sort = bson.D{}

		for _, s := range body.Sorts {
			direction := 1
			if strings.ToLower(s.Direction) == "desc" {
				direction = -1
			}
			sort = append(sort, bson.E{Key: s.Column, Value: direction})
		}
	}

	pipeline = append(pipeline, bson.M{
		"$sort": sort,
	})

	//
	// offset stage
	//
	offset := 0
	if body.Limit != nil {
		offset = *body.Offset
	}

	pipeline = append(pipeline, bson.M{
		"$skip": offset,
	})

	//
	// limit stage
	//
	limit := 10
	if body.Limit != nil {
		limit = *body.Limit
	}

	pipeline = append(pipeline, bson.M{
		"$limit": limit,
	})

	//---------------redis------------------//
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	ifNoneMatch := c.Request().Header.Get("If-None-Match")

	if c.Request().Header.Get("Cache-Control") == "only-if-cached" {

		cacheProducts, found := redisClient.Get(context.Background(), cacheKey).Result()

		if found != nil {
			c.Response().Header().Set("Cache-Control", "no-store")
			c.Response().Header().Set("Connection", "close")
			c.Response().Header().Set("X-Cache-Status", "Miss")
			return c.JSON(http.StatusGatewayTimeout, echo.Map{"message": "The resource is not in the cache, and the server could not retrieve it"})
		}

		var products map[string][]interface{}
		if err := json.Unmarshal([]byte(cacheProducts), &products); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error unmarshal JSON only if cached"})
		}

		maxAgeTime := getMaxAgeTime(c)
		cacheAgeTime, err := redisClient.TTL(context.Background(), cacheKey).Result()
		if err != nil {
			log.Println(err)
		}

		age := int(maxAgeTime) - int(cacheAgeTime.Seconds())
		expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

		cacheEtag := generateETag(cacheProducts)

		if ifNoneMatch != "" && ifNoneMatch == cacheEtag {
			c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
			c.Response().Header().Set("Etag", cacheEtag)
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Hit")

			return c.NoContent(http.StatusNotModified)
		}

		c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
		c.Response().Header().Set("Etag", cacheEtag)
		c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("X-Cache-Status", "Hit")

		return c.JSON(http.StatusOK, products)

	}

	cacheProducts, found := redisClient.Get(context.Background(), cacheKey).Result()
	//cache Hit
	if found == nil {

		var products map[string][]interface{}
		err := json.Unmarshal([]byte(cacheProducts), &products)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}

		maxAgeTime := getMaxAgeTime(c)
		cacheAgeTime, err := redisClient.TTL(context.Background(), cacheKey).Result()
		if err != nil {
			log.Println(err)
		}

		age := int(maxAgeTime) - int(cacheAgeTime.Seconds())
		expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

		cacheEtag := generateETag(cacheProducts)

		if ifNoneMatch != "" && ifNoneMatch == cacheEtag {
			c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
			c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
			c.Response().Header().Set("Etag", cacheEtag)
			c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
			c.Response().Header().Set("X-Cache-Status", "Hit")

			return c.NoContent(http.StatusNotModified)
		}

		c.Response().Header().Set("Age", fmt.Sprintf("%d", age))
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))
		c.Response().Header().Set("Etag", cacheEtag)
		c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
		c.Response().Header().Set("X-Cache-Status", "Hit")

		return c.JSON(http.StatusOK, products)

		// return c.JSON(http.StatusOK, products)

	} else if err != nil {
		log.Println(err)
	}

	//
	// result ผลลัพธ์
	//
	aggServiceUsages, err := h.DB.AggregateServiceUsage(context.Background(), pipeline)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, &models.Exception{
			Status: IntToPointer(http.StatusUnprocessableEntity),
			Detail: "Could not explore service usages, " + err.Error(),
		})
	}

	results := []interface{}{}
	for _, p := range aggServiceUsages {
		results = append(results, p)
	}

	response := &models.Explores{
		Results: results,
	}

	productMarshal, err := json.Marshal(response)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "Error marshaling JSON"})
	}

	maxAgeTime := getMaxAgeTime(c)
	etag := generateETag(string(productMarshal))
	expiresTime := time.Now().Add(time.Duration(maxAgeTime) * time.Second)

	cacheControl := c.Request().Header.Get("Cache-Control")

	if cacheControl == "no-cache" {
		fmt.Println("hrllo world ")
		if clientETag := c.Request().Header.Get("If-None-Match"); clientETag != "" {
			if etag == clientETag {
				c.Response().Header().Set("Cache-Control", "no-cache")
				c.Response().Header().Set("Etag", etag)
				c.Response().Header().Set("X-Cache-Status", "Miss")
				return c.NoContent(http.StatusNotModified)
			}
		}

		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Etag", etag)
		c.Response().Header().Set("X-Cache-Status", "Miss")
		return c.JSON(http.StatusOK, response)
	}

	if cacheControl != "no-store" {
		err = redisClient.Set(context.Background(), cacheKey, productMarshal, time.Duration(maxAgeTime)*time.Second).Err()
		if err != nil {
			log.Println(err)
		}
	}

	if cacheControl == "no-store" {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("X-Cache-Status", "Miss")
		return c.JSON(http.StatusOK, response)
	}

	c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAgeTime))

	c.Response().Header().Set("Etag", etag)
	c.Response().Header().Set("Expires", expiresTime.UTC().Format(http.TimeFormat))
	c.Response().Header().Set("X-Cache-Status", "Miss")

	return c.JSON(http.StatusOK, response)
}
