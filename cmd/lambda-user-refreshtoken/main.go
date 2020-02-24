package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
	"github.com/getsentry/sentry-go"
	user "github.com/retgits/acme-serverless-user"
)

var (
	// AtJwtKey is used to create the Access token signature
	AtJwtKey = []byte("my_secret_key")
	// RtJwtKey is used to create the refresh token signature
	RtJwtKey = []byte("my_secret_key_2")
)

func handleError(area string, headers map[string]string, err error) (events.APIGatewayProxyResponse, error) {
	sentry.CaptureException(fmt.Errorf("error %s: %s", area, err.Error()))
	msg := fmt.Sprintf("error %s: %s", area, err.Error())
	log.Println(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       msg,
		Headers:    headers,
	}, err
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	sentrySyncTransport := sentry.NewHTTPSyncTransport()
	sentrySyncTransport.Timeout = time.Second * 3

	sentry.Init(sentry.ClientOptions{
		Dsn:         os.Getenv("SENTRY_DSN"),
		Transport:   sentrySyncTransport,
		ServerName:  os.Getenv("FUNCTION_NAME"),
		Release:     os.Getenv("VERSION"),
		Environment: os.Getenv("STAGE"),
	})

	headers := request.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	delete(headers, "Content-Length")
	headers["Access-Control-Allow-Origin"] = "*"

	// Create the key attributes
	login, err := user.UnmarshalLoginResponse(request.Body)
	if err != nil {
		return handleError("unmarshalling login", headers, err)
	}

	valid, id, _, err := ValidateToken(login.RefreshToken)

	if !valid || id == "" {
		res := user.VerifyTokenResponse{
			Message: "Invalid Key. User Not Authorized",
			Status:  http.StatusForbidden,
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

	newToken, _ := GenerateAccessToken("eric", id)

	res := user.LoginResponse{
		AccessToken:  newToken,
		RefreshToken: login.RefreshToken,
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

// ValidateToken is used to validate both access_token and refresh_token. It is done based on the "Key ID" provided by the JWT
func ValidateToken(tokenString string) (bool, string, string, error) {

	var key []byte

	var keyID string

	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {

		keyID = token.Header["kid"].(string)
		// If the "kid" (Key ID) is equal to signin_1, then it is compared against access_token secret key, else if it
		// is equal to signin_2 , it is compared against refresh_token secret key.
		if keyID == "signin_1" {
			key = AtJwtKey
		} else if keyID == "signin_2" {
			key = RtJwtKey
		}
		return key, nil
	})

	// Check if signatures are valid.
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			log.Printf("Invalid Token Signature")
			return false, "", keyID, err
		}
		return false, "", keyID, err
	}

	if !token.Valid {
		log.Printf("Invalid Token")
		return false, "", keyID, err
	}

	return true, claims["sub"].(string), keyID, nil
}
