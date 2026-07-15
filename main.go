package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

func main() {
	ctx := context.Background()

	mongodbHost, found := os.LookupEnv("MONGODB_HOST")
	if !found {
		log.Fatal("MONGODB_HOST environment variable is not set")
	}

	mongodbPort, found := os.LookupEnv("MONGODB_PORT")
	if !found {
		log.Fatal("MONGODB_PORT environment variable is not set")
	}

	mongodbUser, found := os.LookupEnv("MONGODB_USER")
	if !found {
		log.Fatal("MONGODB_USER environment variable is not set")
	}

	passwordFile, found := os.LookupEnv("MONGODB_PASSWORD_FILE")
	if !found {
		log.Fatal("MONGODB_PASSWORD_FILE environment variable is not set")
	}

	passwordBytes, err := os.ReadFile(passwordFile)
	if err != nil {
		log.Fatalf("failed to read password file: %v", err)
	}
	mongodbPassword := strings.TrimSpace(string(passwordBytes))

	connectionString := fmt.Sprintf("mongodb://%s:%s", mongodbHost, mongodbPort)

	clientOpts := options.Client().
		ApplyURI(connectionString).
		SetAuth(options.Credential{
			Username: mongodbUser,
			Password: mongodbPassword,
		})

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(context.Background()); err != nil {
			log.Fatalf("failed to disconnect from MongoDB: %v", err)
		}
	}()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatalf("failed to ping MongoDB: %v", err)
	}

	collection := client.Database("test").Collection("test")

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		rootHandler(c, collection)
	})

	r.GET("/healthz", healthHandler)

	r.GET("/logs", func(c *gin.Context) {
		logsHandler(c, collection)
	})

	if err := r.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
