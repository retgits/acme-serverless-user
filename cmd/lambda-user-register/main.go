package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gofrs/uuid"
	user "github.com/retgits/acme-serverless-user"
	"github.com/retgits/acme-serverless-user/internal/datastore/dynamodb"
)

func handleError(area string, err error) (events.APIGatewayProxyResponse, error) {
	msg := fmt.Sprintf("error %s: %s", area, err.Error())
	log.Println(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       msg,
	}, err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	response := events.APIGatewayProxyResponse{}

	// Update the user with an ID
	usr, err := user.UnmarshalUser(request.Body)
	if err != nil {
		return handleError("unmarshalling user", err)
	}
	usr.ID = uuid.Must(uuid.NewV4()).String()

	dynamoStore := dynamodb.New()
	err = dynamoStore.AddUser(usr)
	if err != nil {
		return handleError("getting users", err)
	}

	status := user.RegisterResponse{
		Message:    "User created successfully!",
		ResourceID: usr.ID,
		Status:     http.StatusCreated,
	}

	statusPayload, err := status.Marshal()
	if err != nil {
		return handleError("marshalling response", err)
	}

	response.StatusCode = http.StatusCreated
	response.Body = statusPayload

	return response, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(handler)
}
