package datastore

import user "github.com/retgits/acme-serverless-user"

// Manager ...
type Manager interface {
	GetUser(userID string) (user.User, error)
	FindUser(username string) (user.User, error)
	AllUsers() ([]user.User, error)
	AddUser(usr user.User) error
}
