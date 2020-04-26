/*
  Copyright (C) 2017 Jorge Martinez Hernandez

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package services

import (
	"errors"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

//go:generate mockgen -source=auth.go -destination=auth_mock.go -package=services

type (
	// Auth service interface
	Auth interface {
		// Login a user with username and password
		Login(username, password string) (models.APIKeyPair, error)

		// Register a user with username and password
		Register(username, password string) error

		// Renew access tokens using a refresh token
		Renew(token string) (models.APIKey, error)
	}

	// AuthService implements Auth service for end users
	AuthService struct {
		AuthSecret string
		repo       repo.Users
	}
)

const (
	signingMethod = "HS256"
	refreshType   = "refresh"
)

var (
	// ErrUserUnauthorized signals that a user could not be authenticated
	ErrUserUnauthorized = errors.New("unauthorized Request")

	// ErrUserConflicts signals that a new user name conflicts with an existing one
	ErrUserConflicts = errors.New("username already used")
)

func NewAuthService(authSecret string, userRepo repo.Users) AuthService {
	return AuthService{
		authSecret,
		userRepo,
	}
}

// Login a user
func (a AuthService) Login(username, password string) (models.APIKeyPair, error) {
	user, found := a.repo.UserWithName(username)
	if !found {
		return models.APIKeyPair{}, ErrUserUnauthorized
	}

	if !utils.VerifyPasswordHash(password, user.PasswordHash, user.PasswordSalt) {
		return models.APIKeyPair{}, ErrUserUnauthorized
	}

	keys, err := utils.NewKeyPair(a.AuthSecret, user.ID)
	if err != nil {
		return models.APIKeyPair{}, err
	}

	return keys, nil
}

// Register a user
func (a AuthService) Register(username, password string) error {
	if _, found := a.repo.UserWithName(username); found {
		return ErrUserConflicts
	}

	hash, salt := utils.CreatePasswordHashAndSalt(password)

	user := models.User{
		ID:           utils.CreateID(),
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	a.repo.Create(&user)

	return nil
}

// Renew an API token
func (a AuthService) Renew(token string) (models.APIKey, error) {
	claims, err := utils.ParseJWTClaims(a.AuthSecret, signingMethod, token)
	if err != nil {
		return models.APIKey{}, ErrUserUnauthorized
	}

	if claims["type"].(string) != refreshType {
		return models.APIKey{}, ErrUserUnauthorized
	}

	user, found := a.repo.UserWithID(claims["sub"].(string))
	if !found {
		return models.APIKey{}, ErrUserUnauthorized
	}

	return utils.NewAPIKey(a.AuthSecret, models.AccessKey, user.ID)
}
