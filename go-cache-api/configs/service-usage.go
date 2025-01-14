package configs

import (
	"context"
	"encoding/json"
	"errors"
	"go-cache-api/models"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	Client *mongo.Client
}

const (
	exports = "exports"
)

func Connect(uri string) (*Database, error) {
	ctx, cancle := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancle()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil { 
		return nil, err
	}

	ctx, cancle = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancle()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, err
	}

	return &Database{Client: client}, nil
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

func ChangeKeyId(key string) string {
	keyElems := strings.Split(key, ".")

	for i, k := range keyElems {
		if k == "id" {
			k = "_id"
		}

		keyElems[i] = k
	}

	key = strings.Join(keyElems, ".")

	return key
}

var MongoCommand = map[string]string{
	"and": "$and",
	"or":  "$or",
	"=":   "$eq",
	"!=":  "$ne",
	"<>":  "$ne",
	">":   "$gt",
	">=":  "$gte",
	"<":   "$lt",
	"<=":  "$lte",
}

func FilterToBsonM(filter *models.ExploreFilter) (bson.M, error) {
	var err error

	query := bson.M{}

	ops := []string{}

	logicalOps := []string{"and", "or"}
	compareOps := []string{"=", "!=", "<>", ">", ">=", "<", "<="}

	ops = append(ops, logicalOps...)
	ops = append(ops, compareOps...)

	operator := strings.ToLower(filter.Operator)

	if !IsStringInSlice(operator, ops) {
		return query, errors.New("not support operator '" + operator + "'")
	}

	arguments := filter.Arguments

	if IsStringInSlice(operator, logicalOps) {
		if len(arguments) < 1 {
			return query, errors.New("arguments is invalid, should be at least one object")
		}

		queryList := []bson.M{}

		for _, a := range arguments {
			b, err := json.Marshal(a)
			if err != nil {
				return query, err
			}

			var arg *models.ExploreFilter
			err = json.Unmarshal(b, &arg)
			if err != nil {
				return query, err
			}

			q, err := FilterToBsonM(arg)
			if err != nil {
				return query, err
			}

			queryList = append(queryList, q)
		}

		query[MongoCommand[operator]] = queryList

	} else {

		if len(arguments) != 2 {
			return query, errors.New("arguments is invalid, should be two items in arguments array")
		}

		mapKey, ok := arguments[0].(map[string]interface{})
		if !ok {
			return query, errors.New("arguments[0] is invalid, should be object")
		}

		key, ok := mapKey["property"].(string)
		if !ok {
			return query, errors.New("arguments[0] is invalid")
		}

		//
		// hard code
		//
		var value interface{}

		if IsStringInSlice(key, []string{"time", "createdAt", "updatedAt", "datetime"}) {
			layout := "2006-01-02T15:04:05Z"
			value, err = time.Parse(layout, arguments[1].(string))
			if err != nil {
				return query, errors.New("arguments[1] is invalid")
			}
		} else if IsStringInSlice(key, []string{"userId", "apiKeyId", "createdBy", "updatedBy"}) || strings.Contains(key, ".id") {
			value, err = primitive.ObjectIDFromHex(arguments[1].(string))
			if err != nil {
				return query, errors.New("arguments[1] is invalid")
			}
		} else {
			value = arguments[1]
		}

		//
		// match field id (json) to _id (bson)
		//
		key = ChangeKeyId(key)

		query[key] = bson.M{
			MongoCommand[operator]: value,
		}
	}

	return query, nil
}

func (db *Database) AggregateServiceUsage(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) ([]bson.M, error) {
	collection := GetCollection(ConnectDB(), exports)
	cur, err := collection.Aggregate(ctx, pipeline)
	// defer cur.Close(ctx)
	if err != nil {
		return nil, err
	}

	results := []bson.M{}
	for cur.Next(ctx) {
		var result bson.M
		err := cur.Decode(&result)
		if err != nil {
			return nil, err
		}

		results = append(results, result)

	}

	return results, nil
}
