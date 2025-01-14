package database

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CountServiceUsage is function to count ServiceUsage
func (db *Database) CountServiceUsage(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	collection := db.Client.Database(os.Getenv("API_DB_NAME")).Collection(kindServiceUsages)
	return collection.CountDocuments(ctx, filter, opts...)
}


// AggregateServiceUsage function
func (db *Database) AggregateServiceUsage(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) ([]bson.M, error) {
	collection := db.Client.Database(os.Getenv("API_DB_NAME")).Collection(kindServiceUsages)

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

		fmt.Println(results)
	}
	return results, nil
}
