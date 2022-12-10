package mongo

import (
	"context"
	"log"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Document struct {
	Name  string
	Value SubDocument
	Count int
}
type SubDocument struct {
	Test int
	Ron  string
}

func TestMongo(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://root:example@127.0.0.1:27017/"))
	if err != nil {
		log.Fatal(err)
	}
	db := client.Database("test")
	collection := db.Collection("base")

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cur, err := collection.Aggregate(ctx, mongo.Pipeline{
		bson.D{{"$match", bson.M{}}},
		bson.D{{"$project",
			bson.M{
				"name":       1,
				"value.test": "$value.test",
				"value.ron":  "$value.ron",
				"count": bson.D{{"$size", bson.D{
					{"$cond", bson.A{bson.D{{"$isArray", "$documents"}}, "$documents", bson.A{}}},
				}}},
			}}}})
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var result Document
		err := cur.Decode(&result)
		if err != nil {
			log.Fatal(err)
		}
		// do something with result....
		log.Println(result)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

}
