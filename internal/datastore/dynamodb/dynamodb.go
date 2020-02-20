package dynamodb

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	user "github.com/retgits/acme-serverless-user"
	"github.com/retgits/acme-serverless-user/internal/datastore"
)

// Create a single instance of the dynamoDB service
// which can be reused if the container stays warm
var dbs *dynamodb.DynamoDB

type manager struct{}

// init creates the connection to dynamoDB. If the environment variable
// DYNAMO_URL is set, the connection is made to that URL instead of
// relying on the AWS SDK to provide the URL
func init() {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	if len(os.Getenv("DYNAMO_URL")) > 0 {
		awsSession.Config.Endpoint = aws.String(os.Getenv("DYNAMO_URL"))
	}

	dbs = dynamodb.New(awsSession)
}

// New creates a new datastore manager using Amazon DynamoDB as backend
func New() datastore.Manager {
	return manager{}
}

// GetUser retrieves a single user from DynamoDB based on the userID
func (m manager) GetUser(userID string) (user.User, error) {
	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = USER SK = ID
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("USER"),
	}
	km[":id"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	// Create the QueryInput
	qi := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		KeyConditionExpression:    aws.String("PK = :type AND SK = :id)"),
		ExpressionAttributeValues: km,
	}

	// Execute the DynamoDB query
	qo, err := dbs.Query(qi)
	if err != nil {
		return user.User{}, err
	}

	// Return an error if no user was found
	if len(qo.Items) == 0 {
		return user.User{}, fmt.Errorf("no user found with id %s", userID)
	}

	// Create a user struct from the data
	str := *qo.Items[0]["Payload"].S
	return user.UnmarshalUser(str)
}

// FindUser retrieves a single user from DynamoDB based on the username
func (m manager) FindUser(username string) (user.User, error) {
	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = USER KeyID = ID
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("USER"),
	}
	km[":username"] = &dynamodb.AttributeValue{
		S: aws.String(username),
	}

	// Create the QueryInput
	qi := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		KeyConditionExpression:    aws.String("PK = :type"),
		FilterExpression:          aws.String("KeyID = :username"),
		ExpressionAttributeValues: km,
	}

	// Execute the DynamoDB query
	qo, err := dbs.Query(qi)
	if err != nil {
		return user.User{}, err
	}

	// Return an error if no user was found
	if len(qo.Items) == 0 {
		return user.User{}, fmt.Errorf("no user found with name %s", username)
	}

	// Create a user struct from the data
	str := *qo.Items[0]["Payload"].S
	return user.UnmarshalUser(str)
}

// AllUsers retrieves all users from DynamoDB
func (m manager) AllUsers() ([]user.User, error) {
	// Create a map of DynamoDB Attribute Values containing the table keys
	// for the access pattern PK = USER
	km := make(map[string]*dynamodb.AttributeValue)
	km[":type"] = &dynamodb.AttributeValue{
		S: aws.String("USER"),
	}

	// Create the QueryInput
	qi := &dynamodb.QueryInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		KeyConditionExpression:    aws.String("PK = :type"),
		ExpressionAttributeValues: km,
	}

	qo, err := dbs.Query(qi)
	if err != nil {
		return nil, err
	}

	users := make([]user.User, len(qo.Items))

	for idx, ct := range qo.Items {
		str := *ct["Payload"].S
		usr, err := user.UnmarshalUser(str)
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshalling user data: %s", err.Error()))
			continue
		}
		users[idx] = usr
	}

	return users, nil
}

// AddUser stores a new user in Amazon DynamoDB
func (m manager) AddUser(usr user.User) error {
	// Create a JSON encoded string of the user
	payload, err := usr.Marshal()
	if err != nil {
		return err
	}

	// Create a map of DynamoDB Attribute Values containing the table keys
	km := make(map[string]*dynamodb.AttributeValue)
	km["PK"] = &dynamodb.AttributeValue{
		S: aws.String("USER"),
	}
	km["SK"] = &dynamodb.AttributeValue{
		S: aws.String(usr.ID),
	}

	// Create a map of DynamoDB Attribute Values containing the table data elements
	em := make(map[string]*dynamodb.AttributeValue)
	em[":keyid"] = &dynamodb.AttributeValue{
		S: aws.String(usr.Username),
	}
	em[":payload"] = &dynamodb.AttributeValue{
		S: aws.String(payload),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET Payload = :payload, KeyID = :keyid"),
	}

	_, err = dbs.UpdateItem(uii)
	if err != nil {
		return err
	}

	return nil
}
