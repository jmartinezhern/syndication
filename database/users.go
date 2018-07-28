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

// Package database provides routines to operate on Syndications SQL database
// using models defined in the models package to map data in said database.
package database

import (
	"crypto/rand"
	"io"

	"github.com/varddum/syndication/models"
	"golang.org/x/crypto/scrypt"
)

// AddAPIKey associates an API key with user
func (db *DB) AddAPIKey(key models.APIKey, user models.User) {
	db.db.Model(&user).Association("APIKeys").Append(key)
}

// AddAPIKey associates an API key with user
func AddAPIKey(key models.APIKey, user models.User) {
	defaultInstance.AddAPIKey(key, user)
}

// NewUser creates a new user
func (db *DB) NewUser(username, password string) models.User {
	user := models.User{}
	hash, salt := createPasswordHashAndSalt(password)

	// Construct the user system categories
	unctgAPIID := createAPIID()
	user.Categories = append(user.Categories, models.Category{
		APIID: unctgAPIID,
		Name:  models.Uncategorized,
	})
	user.UncategorizedCategoryAPIID = unctgAPIID

	user.APIID = createAPIID()
	user.PasswordHash = hash
	user.PasswordSalt = salt
	user.Username = username

	db.db.Create(&user).Related(&user.Categories)

	return user
}

// NewUser creates a new user
func NewUser(username, password string) models.User {
	return defaultInstance.NewUser(username, password)
}

// DeleteUser with API ID
func (db *DB) DeleteUser(apiID string) error {
	user, found := db.UserWithAPIID(apiID)
	if !found {
		return ErrModelNotFound
	}

	db.db.Delete(&user)
	return nil
}

// DeleteUser with API ID
func DeleteUser(apiID string) error {
	return defaultInstance.DeleteUser(apiID)
}

// ChangeUserName for user with userID
func (db *DB) ChangeUserName(id, newName string) error {
	user, found := db.UserWithAPIID(id)
	if !found {
		return ErrModelNotFound
	}

	db.db.Model(&user).Update("username", newName)
	return nil
}

// ChangeUserName for user with id
func ChangeUserName(id, newName string) error {
	return defaultInstance.ChangeUserName(id, newName)
}

// ChangeUserPassword for user with id
func (db *DB) ChangeUserPassword(id, newPassword string) error {
	user, found := db.UserWithAPIID(id)
	if !found {
		return ErrModelNotFound
	}

	hash, salt := createPasswordHashAndSalt(newPassword)

	db.db.Model(&user).Update(models.User{
		PasswordHash: hash,
		PasswordSalt: salt,
	})

	return nil
}

// ChangeUserPassword for user with id
func ChangeUserPassword(id, newPassword string) error {
	return defaultInstance.ChangeUserPassword(id, newPassword)
}

// Users returns a list of all User entries.
// The parameter fields provides a way to select
// which fields are populated in the returned models.
func (db *DB) Users(fields ...string) []models.User {
	users := []models.User{}

	selectFields := "id,api_id"
	if len(fields) != 0 {
		for _, field := range fields {
			selectFields = selectFields + "," + field
		}
	}
	db.db.Select(selectFields).Find(&users)

	return users
}

// Users returns a list of all User entries.
// The parameter fields provides a way to select
// which fields are populated in the returned models.
func Users(fields ...string) []models.User {
	return defaultInstance.Users(fields...)
}

// UserWithName returns a User with username
func (db *DB) UserWithName(username string) (user models.User, found bool) {
	found = !db.db.First(&user, "username = ?", username).RecordNotFound()
	return
}

// UserWithName returns a User with username
func UserWithName(username string) (models.User, bool) {
	return defaultInstance.UserWithName(username)
}

// UserWithCredentials returns a user whose credentials matches the ones given.
// Ok will be false if the user was not found or if the credentials did not match.
// This function assumes that the password does not exceed scrypt's payload size.
func (db *DB) UserWithCredentials(username, password string) (models.User, bool) {
	foundUser, ok := db.UserWithName(username)
	if !ok {
		return models.User{}, ok
	}

	hash, err := scrypt.Key([]byte(password), foundUser.PasswordSalt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		return models.User{}, false
	}

	for i, hashByte := range hash {
		if hashByte != foundUser.PasswordHash[i] {
			return models.User{}, false
		}
	}

	return foundUser, true
}

// UserWithCredentials returns a user whose credentials matches the ones given.
// Ok will be false if the user was not found or if the credentials did not match.
// This function assumes that the password does not exceed scrypt's payload size.
func UserWithCredentials(username, password string) (models.User, bool) {
	return defaultInstance.UserWithCredentials(username, password)
}

// UserWithAPIID returns a User with id
func (db *DB) UserWithAPIID(apiID string) (user models.User, found bool) {
	found = !db.db.First(&user, "api_id = ?", apiID).RecordNotFound()
	return
}

// UserWithAPIID returns a User with id
func UserWithAPIID(apiID string) (models.User, bool) {
	return defaultInstance.UserWithAPIID(apiID)
}

// KeyBelongsToUser returns true if the given APIKey is owned by user
func (db *DB) KeyBelongsToUser(key models.APIKey, user models.User) bool {
	return !db.db.Model(&user).Where("key = ?", key.Key).Related(&key).RecordNotFound()
}

// KeyBelongsToUser returns true if the given APIKey is owned by user
func KeyBelongsToUser(key models.APIKey, user models.User) bool {
	return defaultInstance.KeyBelongsToUser(key, user)
}

func createPasswordHashAndSalt(password string) ([]byte, []byte) {
	var err error

	salt := make([]byte, PWSaltBytes)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err) // We must be able to read from random
	}

	hash, err := scrypt.Key([]byte(password), salt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		panic(err) // We must never get an error
	}

	return hash, salt
}
