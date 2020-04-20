// Package mongodb leverages cross-platform document-oriented database program. Classified as a
// NoSQL database program, MongoDB uses JSON-like documents with schema.
package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/retgits/acme-serverless-user/internal/datastore"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The pointer to MongoDB provides the API operation methods for making requests to MongoDB.
// This specifically creates a single instance of the MongoDB service which can be reused if the
// container stays warm.
var dbs *mongo.Collection

// manager is an empty struct that implements the methods of the
// Manager interface.
type manager struct{}

// init creates the connection to MongoDB.
func init() {
	username := os.Getenv("MONGO_USERNAME")
	password := os.Getenv("MONGO_PASSWORD")
	hostname := os.Getenv("MONGO_HOSTNAME")
	port := os.Getenv("MONGO_PORT")

	connString := fmt.Sprintf("mongodb+srv://%s:%s@%s:%s", username, password, hostname, port)
	if strings.HasSuffix(connString, ":") {
		connString = connString[:len(connString)-1]
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connString))
	if err != nil {
		log.Fatalf("error connecting to MongoDB: %s", err.Error())
	}
	dbs = client.Database("acmeserverless").Collection("user")
}

// New creates a new datastore manager using Amazon DynamoDB as backend
func New() datastore.Manager {
	return manager{}
}

// GetUser retrieves a single user from MongoDB based on the userID
func (m manager) GetUser(userID string) (acmeserverless.User, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	res := dbs.FindOne(ctx, bson.D{{"SK", userID}})

	raw, err := res.DecodeBytes()
	if err != nil {
		return acmeserverless.User{}, fmt.Errorf("unable to decode bytes: %s", err.Error())
	}

	payload := raw.Lookup("Payload").StringValue()
	return acmeserverless.UnmarshalUser(payload)
}

// FindUser retrieves a single user from DynamoDB based on the username
func (m manager) FindUser(username string) (acmeserverless.User, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	res := dbs.FindOne(ctx, bson.D{{"KeyID", username}})

	raw, err := res.DecodeBytes()
	if err != nil {
		return acmeserverless.User{}, fmt.Errorf("unable to decode bytes: %s", err.Error())
	}

	payload := raw.Lookup("Payload").StringValue()

	// Return an error if no user was found
	if len(payload) < 5 {
		return acmeserverless.User{}, fmt.Errorf("no user found with name %s", username)
	}

	// Create a user struct from the data
	return acmeserverless.UnmarshalUser(payload)
}

// AllUsers retrieves all users from DynamoDB
func (m manager) AllUsers() ([]acmeserverless.User, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	cursor, err := dbs.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	var results []bson.M

	if err = cursor.All(ctx, &results); err != nil {
		log.Fatal(err)
	}

	users := make([]acmeserverless.User, len(results))

	for idx, ct := range results {
		usr, err := acmeserverless.UnmarshalUser(ct["Payload"].(string))
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshalling user data: %s", err.Error()))
			continue
		}
		users[idx] = usr
	}

	return users, nil
}

// AddUser stores a new user in Amazon DynamoDB
func (m manager) AddUser(usr acmeserverless.User) error {
	payload, err := usr.Marshal()
	if err != nil {
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	_, err = dbs.InsertOne(ctx, bson.D{{"SK", usr.ID}, {"KeyID", usr.Username}, {"PK", "USER"}, {"Payload", string(payload)}})

	return err
}
