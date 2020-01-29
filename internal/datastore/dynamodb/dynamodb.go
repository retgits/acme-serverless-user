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

type manager struct{}

func New() datastore.Manager {
	return manager{}
}

func (m manager) GetUser(userID string) (user.User, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	// Create a map of DynamoDB Attribute Values containing the table keys
	km := make(map[string]*dynamodb.AttributeValue)
	km[":userid"] = &dynamodb.AttributeValue{
		S: aws.String(userID),
	}

	si := &dynamodb.ScanInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		ExpressionAttributeValues: km,
		FilterExpression:          aws.String("ID = :userid"),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		return user.User{}, err
	}

	if len(so.Items) == 0 {
		return user.User{}, fmt.Errorf("no user found with id %s", userID)
	}

	str := *so.Items[0]["UserContent"].S
	usr, err := user.UnmarshalUser(str)
	if err != nil {
		return user.User{}, err
	}
	return usr, nil
}

func (m manager) FindUser(username string) (user.User, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	// Create a map of DynamoDB Attribute Values containing the table keys
	km := make(map[string]*dynamodb.AttributeValue)
	km[":username"] = &dynamodb.AttributeValue{
		S: aws.String(username),
	}

	si := &dynamodb.ScanInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		ExpressionAttributeValues: km,
		FilterExpression:          aws.String("UserName = :username"),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		return user.User{}, err
	}

	if len(so.Items) == 0 {
		return user.User{}, fmt.Errorf("no user found with id %s", username)
	}

	str := *so.Items[0]["UserContent"].S
	usr, err := user.UnmarshalUser(str)
	if err != nil {
		return user.User{}, err
	}
	return usr, nil
}

func (m manager) AllUsers() ([]user.User, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	si := &dynamodb.ScanInput{
		TableName: aws.String(os.Getenv("TABLE")),
	}

	so, err := dbs.Scan(si)
	if err != nil {
		return nil, err
	}

	users := make([]user.User, len(so.Items))

	for idx, ct := range so.Items {
		str := *ct["UserContent"].S
		usr, err := user.UnmarshalUser(str)
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshalling user data: %s", err.Error()))
			break
		}
		users[idx] = usr
	}

	return users, nil
}

func (m manager) AddUser(usr user.User) error {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("REGION")),
	}))

	dbs := dynamodb.New(awsSession)

	// Marshal the newly updated user struct
	payload, err := usr.Marshal()
	if err != nil {
		return err
	}

	// Create a map of DynamoDB Attribute Values containing the table keys
	km := make(map[string]*dynamodb.AttributeValue)
	km["ID"] = &dynamodb.AttributeValue{
		S: aws.String(usr.ID),
	}

	em := make(map[string]*dynamodb.AttributeValue)
	em[":content"] = &dynamodb.AttributeValue{
		S: aws.String(payload),
	}
	em[":username"] = &dynamodb.AttributeValue{
		S: aws.String(usr.Username),
	}

	uii := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(os.Getenv("TABLE")),
		Key:                       km,
		ExpressionAttributeValues: em,
		UpdateExpression:          aws.String("SET UserContent = :content, UserName = :username"),
	}

	_, err = dbs.UpdateItem(uii)
	if err != nil {
		return err
	}

	return nil
}
