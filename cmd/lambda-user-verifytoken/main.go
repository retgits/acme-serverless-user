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
	acmeserverless "github.com/retgits/acme-serverless"
	wflambda "github.com/wavefronthq/wavefront-lambda-go"
)

// handler handles the API Gateway events and returns an error if anything goes wrong.
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Initiialize a connection to Sentry to capture errors and traces
	sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("SENTRY_DSN"),
		Transport: &sentry.HTTPSyncTransport{
			Timeout: time.Second * 3,
		},
		ServerName:  os.Getenv("FUNCTION_NAME"),
		Release:     os.Getenv("VERSION"),
		Environment: os.Getenv("STAGE"),
	})

	// Create headers if they don't exist and add
	// the CORS required headers, otherwise the response
	// will not be accepted by browsers.
	headers := request.Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Access-Control-Allow-Origin"] = "*"

	// Create the key attributes
	login, err := acmeserverless.UnmarshalLoginResponse(request.Body)
	if err != nil {
		return handleError("unmarshalling login", headers, err)
	}

	valid, _, key, err := ValidateToken(login.AccessToken)

	res := acmeserverless.VerifyTokenResponse{
		Message: "Token Valid. User Authorized",
		Status:  http.StatusOK,
	}

	if !valid || key != "signin_1" {
		res.Message = "Invalid Key. User Not Authorized"
		res.Status = http.StatusForbidden
	}

	payload, err := res.Marshal()
	if err != nil {
		return handleError("marshalling response", headers, err)
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(payload),
		Headers:    headers,
	}

	return response, nil
}

// handleError takes the activity where the error occured and the error object and sends a message to sentry.
// The original error, together with the appropriate API Gateway Proxy Response, is returned so it can be thrown.
func handleError(area string, headers map[string]string, err error) (events.APIGatewayProxyResponse, error) {
	sentry.CaptureException(fmt.Errorf("error %s: %s", area, err.Error()))
	msg := fmt.Sprintf("error %s: %s", area, err.Error())
	log.Println(msg)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusBadRequest,
		Body:       msg,
		Headers:    headers,
	}, nil
}

// The main method is executed by AWS Lambda and points to the handler
func main() {
	lambda.Start(wflambda.Wrapper(handler))
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
			key = []byte("my_secret_key")
		} else if keyID == "signin_2" {
			key = []byte("my_secret_key_2")
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
