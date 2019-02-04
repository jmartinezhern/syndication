package usecases

import (
	"errors"
	"github.com/jmartinezhern/syndication/models"

	"github.com/jmartinezhern/syndication/database"
)

type (
	// Admin interface defines the Admin usecases
	Admin interface {
		// NewUser creates a new user with user name and password
		NewUser(username, password string) (models.User, error)

		// DeleteUser with id
		DeleteUser(id string) error

		//User with id
		User(id string) (models.User, bool)

		// Users gets a list of users
		Users() []models.User

		// ChangeUserName of user with id
		ChangeUserName(id, name string) error

		// ChangeUserPassword for user with id
		ChangeUserPassword(id, password string) error
	}

	// AdminUsecase implement the Admin interface
	AdminUsecase struct{}
)

var (
	// ErrUsernameConflicts signals that a username exists in the database
	ErrUsernameConflicts = errors.New("Username already exists")

	// ErrUserNotFound signals that a user could not be found
	ErrUserNotFound = errors.New("User not found")
)

// NewUser creates a new user
func (a *AdminUsecase) NewUser(username, password string) (models.User, error) {
	if _, found := database.UserWithName(username); found {
		return models.User{}, ErrUsernameConflicts
	}

	return database.NewUser(username, password), nil
}

// DeleteUser deletes a user with userID
func (a *AdminUsecase) DeleteUser(id string) error {
	return database.DeleteUser(id)
}

// User gets a user with id
func (a *AdminUsecase) User(id string) (models.User, bool) {
	return database.UserWithAPIID(id)
}

// Users returns all users
func (a *AdminUsecase) Users() []models.User {
	return database.Users("username,id")
}

// ChangeUserName for user with id
func (a *AdminUsecase) ChangeUserName(id string, name string) error {
	return database.ChangeUserName(id, name)
}

// ChangeUserPassword for user with id
func (a *AdminUsecase) ChangeUserPassword(id string, password string) error {
	return database.ChangeUserPassword(id, password)
}
