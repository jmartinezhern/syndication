package services

import (
	"errors"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	// Admin interface defines the Admin services
	Admin interface {
		RegisterInitialAdmin(username, password string) models.Admin

		// NewUser creates a new user with user name and password
		NewUser(username, password string) (models.User, error)

		// DeleteUser with id
		DeleteUser(id string) error

		// User with id
		User(id string) (models.User, bool)

		// Users gets a list of users
		Users(continuationID string, count int) ([]models.User, string)
	}

	// AdminServices implement the Admin interface
	AdminServices struct {
		adminsRepo repo.Admins
		usersRepo  repo.Users
	}
)

var (
	// ErrUsernameConflicts signals that a username exists in the tagsRepo
	ErrUsernameConflicts = errors.New("username already exists")

	// ErrUserNotFound signals that a user could not be found
	ErrUserNotFound = errors.New("user not found")
)

func NewAdminsServices(adminsRepo repo.Admins, usersRepo repo.Users) AdminServices {
	return AdminServices{
		adminsRepo,
		usersRepo,
	}
}

func (a AdminServices) RegisterInitialAdmin(username, password string) models.Admin {
	if _, found := a.adminsRepo.InitialUser(); found {
		// TODO: probably error if we have already registered
		return models.Admin{}
	}

	hash, salt := utils.CreatePasswordHashAndSalt(password)

	admin := models.Admin{
		APIID:        utils.CreateAPIID(),
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	a.adminsRepo.Create(&admin)

	// TODO: probably don't need to do this
	return admin
}

// NewUser creates a new user
func (a *AdminServices) NewUser(username, password string) (models.User, error) {
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
func (a *AdminServices) DeleteUser(id string) error {
	return a.usersRepo.Delete(id)
}

// User gets a user with id
func (a *AdminServices) User(id string) (models.User, bool) {
	return a.usersRepo.UserWithID(id)
}

// Users returns all users
func (a *AdminServices) Users(continuationID string, count int) (users []models.User, next string) {
	return a.usersRepo.List(continuationID, count)
}
