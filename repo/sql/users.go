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

// Package repo provides routines to operate on Syndications SQL repo
// using models defined in the models package to map data in said repo.
package sql

import (
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
)

type (
	Users struct {
		db *DB
	}
)

func NewUsers(db *DB) Users {
	return Users{
		db,
	}
}

// Create a new user
func (u Users) Create(user *models.User) {
	u.db.db.Create(user)
}

// Update a user
func (u Users) Update(user *models.User) error {
	dbUser, found := u.UserWithID(user.APIID)
	if !found {
		return repo.ErrModelNotFound
	}

	u.db.db.Model(&dbUser).Updates(user).RecordNotFound()
	return nil
}

// UserWithID returns a User with id
func (u Users) UserWithID(id string) (user models.User, found bool) {
	found = !u.db.db.First(&user, "api_id = ?", id).RecordNotFound()
	return
}

// Delete a user
func (u Users) Delete(id string) error {
	user, found := u.UserWithID(id)
	if !found {
		return repo.ErrModelNotFound
	}

	u.db.db.Delete(user)
	return nil
}

// List all users
func (u Users) List(continuationID string, count int) (users []models.User, next string) {
	query := u.db.db.Limit(count + 1)

	if continuationID != "" {
		user, found := u.UserWithID(continuationID)
		if found {
			query = query.Where("id >= ?", user.ID)
		}
	}
	query.Find(&users)

	if len(users) > count {
		next = users[len(users)-1].APIID
		users = users[:len(users)-1]
	}

	return
}

// UserWithName returns a User with username
func (u Users) UserWithName(name string) (user models.User, found bool) {
	found = !u.db.db.First(&user, "username = ?", name).RecordNotFound()
	return
}

// OwnsKey returns true if the given APIKey is owned by user
func (u Users) OwnsKey(key *models.APIKey, user *models.User) bool {
	return !u.db.db.Model(user).Where("key = ?", key.Key).Related(key).RecordNotFound()
}

// AddAPIKey associates an API key with user
func (u Users) AddAPIKey(key *models.APIKey, user *models.User) {
	u.db.db.Model(user).Association("APIKeys").Append(key)
}
