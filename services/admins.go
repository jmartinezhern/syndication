package services

import (
	"errors"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	// Users interface defines the Users services
	Users interface {
		// NewUser creates a new user with user name and password
		NewUser(username, password string) (models.User, error)

		// DeleteUser with id
		DeleteUser(id string) error

		// User with id
		User(id string) (models.User, bool)

		// Users gets a list of users
		Users(continuationID string, count int) ([]models.User, string)
	}

	// UsersService implement the Users interface
	UsersService struct {
		usersRepo repo.Users
	}
)

var (
	// ErrUsernameConflicts signals that a username exists in the tagsRepo
	ErrUsernameConflicts = errors.New("username already exists")

	// ErrUserNotFound signals that a user could not be found
	ErrUserNotFound = errors.New("user not found")
)

func NewUsersService(usersRepo repo.Users) UsersService {
	return UsersService{
		usersRepo,
	}
}

// NewUser creates a new user
func (a *UsersService) NewUser(username, password string) (models.User, error) {
	if _, found := a.usersRepo.UserWithName(username); found {
		return models.User{}, ErrUsernameConflicts
	}

	hash, salt := utils.CreatePasswordHashAndSalt(password)

	user := models.User{
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	a.usersRepo.Create(&user)

	return user, nil
}

// DeleteUser deletes a user with userID
func (a *UsersService) DeleteUser(id string) error {
	return a.usersRepo.Delete(id)
}

// User gets a user with id
func (a *UsersService) User(id string) (models.User, bool) {
	return a.usersRepo.UserWithID(id)
}

// Users returns all users
func (a *UsersService) Users(continuationID string, count int) (users []models.User, next string) {
	return a.usersRepo.List(continuationID, count)
}
