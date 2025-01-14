package configs

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ConnectDB() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(EnvMongoURI()))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	return client
}



func GetCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database("project-api-cache").Collection(collectionName)
	return collection
}



func ConnectRedis() *redis.Client {
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB:   0,
	})

	_, err := redisClient.Ping(ctx).Result()
    if err != nil {
        log.Fatalln("Redis connection was refused", err)
    }

	return redisClient
}










