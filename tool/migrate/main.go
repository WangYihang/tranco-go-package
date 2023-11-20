package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jessevdk/go-flags"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Options struct {
	InputFilePath string `short:"i" long:"input-filepath" description:"input filepath" required:"true"`
}

var cliOptions = Options{}

func init() {
	// Parse flags
	_, err := flags.ParseArgs(&cliOptions, os.Args)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}
	}()

	db := client.Database("domain")
	collection := db.Collection("rank")

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"name": "google.com"},
		bson.M{
			"$set": bson.M{
				"tranco_ranks": bson.M{
					"2019-01-01": bson.M{
						"rank": 1,
						"date": "2019-01-01",
					},
				},
			},
		},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		log.Fatalf("Failed to update document: %v", err)
	}
}
