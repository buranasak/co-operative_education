package database

import (
	"context"
	"encoding/json"
	"errors"
	"explore-api/model"
	"explore-api/tool"

	"net/url"
	"strconv"
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

// collection list of database. 
const (
	kindServices        = "services"
	kindServiceUsages   = "serviceUsages"
)

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

//ต่อ mongodb
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

// OptionSortBson function 
func OptionSortBson(value string) (bson.D, error) {
	order := bson.D{}

	if strings.ToLower(strings.TrimSpace(value)) == "" {
		return order, errors.New("wrong format")
	}

	for _, qo := range strings.Split(value, ",") {
		qo, _ = url.QueryUnescape(qo)
		qo = strings.ReplaceAll(qo, "+", " ")
		qoElems := strings.Split(qo, ":")

		if len(qoElems) > 2 {
			return order, errors.New("wrong format")
		}

		key := strings.TrimSpace(qoElems[0])

		if key == "`id`" {
			key = "_id"
		}

		direction := 1
		if len(qoElems) == 2 {
			elemValue := strings.ToLower(strings.TrimSpace(qoElems[1]))
			if !tool.IsStringInSlice(elemValue, []string{"asc", "desc"}) {
				return order, errors.New("wrong direction, should be asc, desc or blank (default asc)")
			}

			if elemValue == "desc" {
				direction = -1
			}
		}

		order = append(order, primitive.E{Key: key, Value: direction})
	}

	return order, nil
}



func GenerateFilterBson(queryParams url.Values, ignorequeryParams []string) []bson.M {
	and := []bson.M{}

	for key, values := range queryParams {
		if tool.IsStringInSlice(key, ignorequeryParams) {
			continue
		}

		// check id of object json bson
		key = strings.ReplaceAll(key, ".id", "._id")

		or := []bson.M{}
		for _, v := range values {
			valueSlices := strings.Split(v, ",")

			if len(valueSlices) > 1 {
				valueSlices = append(valueSlices, v)
			}

			for _, vs := range valueSlices {
				vs = strings.TrimSpace(vs)

				if string(vs[0]) == "*" && string(vs[len(vs)-1]) == "*" {
					vs = strings.ReplaceAll(vs, "*", "")
					or = append(or, bson.M{key: primitive.Regex{Pattern: vs, Options: "i"}})
				} else if strings.Contains(vs, "*") {
					vs = strings.ReplaceAll(vs, "*", ".")
					or = append(or, bson.M{key: primitive.Regex{Pattern: vs, Options: "i"}})
				} else {
					or = append(or, bson.M{key: vs})

					if f, err := strconv.ParseFloat(vs, 64); err == nil {
						or = append(or, bson.M{key: f})
					}

					if o, err := primitive.ObjectIDFromHex(vs); err == nil {
						or = append(or, bson.M{key: o})
					}

					if b, err := strconv.ParseBool(vs); err == nil {
						or = append(or, bson.M{key: b})
					}
				}
			}
		}

		and = append(and, bson.M{"$or": or})
	}

	return and
}





//filter เพื่อ query ข้อมูลใน mongo
func FilterToBsonM(filter *model.ExploreFilter) (bson.M, error) {
	var err error

	query := bson.M{}

	ops := []string{}
	
	logicalOps := []string{"and", "or"} 
	compareOps := []string{"=", "!=", "<>", ">", ">=", "<", "<="}

	ops = append(ops, logicalOps...)
	ops = append(ops, compareOps...)

	operator := strings.ToLower(filter.Operator)

	if !tool.IsStringInSlice(operator, ops) {
		return query, errors.New("not support operator '" + operator + "'")
	}

	arguments := filter.Arguments

	if tool.IsStringInSlice(operator, logicalOps) {
		if len(arguments) < 1 {
			return query, errors.New("arguments is invalid, should be at least one object")
		}

		queryList := []bson.M{}

		for _, a := range arguments {
			b, err := json.Marshal(a)
			if err != nil {
				return query, err
			}

			var arg *model.ExploreFilter
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

		if tool.IsStringInSlice(key, []string{"time", "createdAt", "updatedAt", "datetime"}) {
			layout := "2006-01-02T15:04:05Z"
			value, err = time.Parse(layout, arguments[1].(string))
			if err != nil {
				return query, errors.New("arguments[1] is invalid")
			}
		} else if tool.IsStringInSlice(key, []string{"userId", "apiKeyId", "createdBy", "updatedBy"}) || strings.Contains(key, ".id") {
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
