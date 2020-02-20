package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	user "github.com/retgits/acme-serverless-user"
	"github.com/retgits/acme-serverless-user/internal/datastore/dynamodb"
)

func handleError(area string, headers map[string]string, err error) (events.APIGatewayProxyResponse, error) {
	msg := fmt.Sprintf("error %s: %s", area, err.Error())
	log.Println(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       msg,
		Headers:    headers,
	}, err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := request.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	delete(headers, "Content-Length")
	headers["Access-Control-Allow-Origin"] = "*"

	dynamoStore := dynamodb.New()
	users, err := dynamoStore.AllUsers()
	if err != nil {
		return handleError("getting users", headers, err)
	}

	res := user.AllUsers{
		Data: users,
	}

	payload, err := res.Marshal()
	if err != nil {
		return handleError("marshalling response", headers, err)
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       payload,
		Headers:    headers,
	}

	return response, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(handler)
}
