package main

import (
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var (
	secretkey1 = []byte("my_secret_key")
	secretkey2 = []byte("my_secret_key_2")
)

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

	refreshTokenString, err := refreshToken.SignedString(secretkey2)
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
	tokenString, err := token.SignedString(secretkey1)
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
			key = secretkey1
		} else if keyID == "signin_2" {
			key = secretkey2
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
