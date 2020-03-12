// Package datastore contains the interfaces that the User service
// in the ACME Serverless Fitness Shop needs to store and retrieve data.
// In order to add a new service, the Manager interface
// needs to be implemented.
package datastore

import user "github.com/retgits/acme-serverless-user"

// Manager is the interface that describes the methods the
// data store needs to implement to be able to work with
// the ACME Serverless Fitness Shop.
type Manager interface {
	GetUser(userID string) (user.User, error)
	FindUser(username string) (user.User, error)
	AllUsers() ([]user.User, error)
	AddUser(usr user.User) error
}
