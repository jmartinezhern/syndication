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
	"github.com/jmartinezhern/syndication/models"
)

// AddAPIKey associates an API key with user
func (db *DB) AddAPIKey(key models.APIKey, user models.User) {
	db.db.Model(&user).Association("APIKeys").Append(key)
}

// AddAPIKey associates an API key with user
func AddAPIKey(key models.APIKey, user models.User) {
	defaultInstance.AddAPIKey(key, user)
}

// CreateUser creates a new user
func (db *DB) CreateUser(user *models.User) {
	db.db.Create(user)
}

// CreateUser creates a new user
func CreateUser(user *models.User) {
	defaultInstance.CreateUser(user)
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

// UpdateUser object
func (db *DB) UpdateUser(user *models.User) {
	db.db.Save(user)
}

// UpdateUser object
func UpdateUser(user *models.User) {
	defaultInstance.UpdateUser(user)
}

// Users returns a list of users
func (db *DB) Users() []models.User {
	users := []models.User{}

	db.db.Find(&users)

	return users
}

// Users returns a list of users
func Users() []models.User {
	return defaultInstance.Users()
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
