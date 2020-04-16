// Package user contains all events that the User service
// in the ACME Serverless Fitness Shop can send and receive.
package user

var (
	// ATJWTKey is used to create the Access token signature
	ATJWTKey = []byte("my_secret_key")
	// RTJWTKey is used to create the refresh token signature
	RTJWTKey = []byte("my_secret_key_2")
)
