package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
	user "github.com/retgits/acme-serverless-user"
	"github.com/retgits/acme-serverless-user/internal/datastore/dynamodb"
)

var (
	// AtJwtKey is used to create the Access token signature
	AtJwtKey = []byte("my_secret_key")
	// RtJwtKey is used to create the refresh token signature
	RtJwtKey = []byte("my_secret_key_2")
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

	// Create the key attributes
	usr, err := user.UnmarshalUser(request.Body)
	if err != nil {
		return handleError("unmarshalling user", headers, err)
	}

	dynamoStore := dynamodb.New()
	usr, err = dynamoStore.FindUser(usr.Username)
	if err != nil {
		return handleError("getting users", headers, err)
	}

	accessToken, refreshToken, err := GenerateTokenPair(usr.Username, usr.ID)
	if err != nil {
		return handleError("generating accesstoken", headers, err)
	}

	res := user.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Status:       http.StatusOK,
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

// GenerateTokenPair creates and returns a new set of access_token and refresh_token.
func GenerateTokenPair(username string, uuid string) (string, string, error) {

	tokenString, err := GenerateAccessToken(username, uuid)
	if err != nil {
		return "", "", err
	}

	// Create Refresh token, this will be used to get new access token.
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	refreshToken.Header["kid"] = "signin_2"

	expirationTimeRefreshToken := time.Now().Add(15 * time.Minute).Unix()

	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = uuid
	rtClaims["exp"] = expirationTimeRefreshToken

	refreshTokenString, err := refreshToken.SignedString(RtJwtKey)
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}

func GenerateAccessToken(username string, uuid string) (string, error) {
	// Declare the expiration time of the access token
	// Here the expiration is 5 minutes
	expirationTimeAccessToken := time.Now().Add(5 * time.Minute).Unix()

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.New(jwt.SigningMethodHS256)
	token.Header["kid"] = "signin_1"
	claims := token.Claims.(jwt.MapClaims)
	claims["Username"] = username
	claims["exp"] = expirationTimeAccessToken
	claims["sub"] = uuid

	// Create the JWT string
	tokenString, err := token.SignedString(AtJwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
