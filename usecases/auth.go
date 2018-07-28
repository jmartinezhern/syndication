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

package usecases

import (
	"errors"
	"time"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"

	"github.com/dgrijalva/jwt-go"
)

type (
	// Auth usecase interface
	Auth interface {
		// Authenticate a user
		Authenticate(token jwt.Token) (models.User, bool)
		Login(username, password string) (models.APIKeyPair, error)
		Register(username, password string) (models.APIKeyPair, error)
		Renew(token string) (models.APIKey, error)
	}

	// AuthUsecase implements Auth usecase
	AuthUsecase struct{ AuthSecret string }
)

const (
	signingMethod = "HS256"
)

var (
	// ErrUserUnauthorized signals that a user could not be authenticated
	ErrUserUnauthorized = errors.New("Unauthorized Request")

	// ErrUserConflicts signals that a new user name conflicts with an existing one
	ErrUserConflicts = errors.New("Username already used")
)

// Authenticate a user
func (a AuthUsecase) Authenticate(token jwt.Token) (models.User, bool) {
	claims := token.Claims.(jwt.MapClaims)

	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return models.User{}, false
	}

	user, found := database.UserWithAPIID(claims["id"].(string))
	if !found {
		return models.User{}, false
	}

	// Check that a refresh key was not used to authenticate
	found = database.KeyBelongsToUser(models.APIKey{
		Key: token.Raw,
	}, user)
	if found {
		return models.User{}, false
	}

	return user, true
}

const (
	refreshKeyExpirationInterval = time.Hour * 24 * 7
	accessKeyExpirationInterval  = time.Hour * 24 * 3
)

// Login a user
func (a AuthUsecase) Login(username, password string) (models.APIKeyPair, error) {
	user, found := database.UserWithCredentials(username, password)
	if !found {
		return models.APIKeyPair{}, ErrUserUnauthorized
	}

	return a.createNewKeyPair(user)
}

// Register a user
func (a AuthUsecase) Register(username, password string) (models.APIKeyPair, error) {
	if _, found := database.UserWithName(username); found {
		return models.APIKeyPair{}, ErrUserConflicts
	}

	user := database.NewUser(username, password)

	return a.createNewKeyPair(user)
}

// Renew an API token
func (a AuthUsecase) Renew(token string) (models.APIKey, error) {
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// Check the signing method
		if t.Method.Alg() != signingMethod {
			return nil, errors.New("jwt signing methods mismatch")
		}
		return []byte(a.AuthSecret), nil
	})
	if err != nil {
		return models.APIKey{}, ErrUserUnauthorized
	}

	claims := jwtToken.Claims.(jwt.MapClaims)
	user, found := database.UserWithAPIID(claims["id"].(string))
	if !found {
		return models.APIKey{}, ErrUserUnauthorized
	}

	key := models.APIKey{
		Key: token,
	}

	if !database.KeyBelongsToUser(key, user) {
		return models.APIKey{}, ErrUserUnauthorized
	}

	return newAPIKey(a.AuthSecret, models.AccessKey, user)
}

func (a AuthUsecase) createNewKeyPair(user models.User) (models.APIKeyPair, error) {
	accessKey, err := newAPIKey(a.AuthSecret, models.AccessKey, user)
	if err != nil {
		return models.APIKeyPair{}, err
	}

	refreshKey, err := newAPIKey(a.AuthSecret, models.RefreshKey, user)
	if err != nil {
		return models.APIKeyPair{}, err
	}

	database.AddAPIKey(refreshKey, user)

	return models.APIKeyPair{
		AccessKey:  accessKey.Key,
		RefreshKey: refreshKey.Key,
	}, nil
}

func newAPIKey(secret string, keyType models.APIKeyType, user models.User) (models.APIKey, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = user.APIID

	switch keyType {
	case models.RefreshKey:
		claims["exp"] = time.Now().Add(refreshKeyExpirationInterval).Unix()
	case models.AccessKey:
		claims["exp"] = time.Now().Add(accessKeyExpirationInterval).Unix()
	}

	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return models.APIKey{}, err
	}

	return models.APIKey{
		Key:  t,
		Type: keyType,
	}, nil
}
