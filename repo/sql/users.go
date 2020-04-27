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
	"github.com/jinzhu/gorm"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
)

type (
	Users struct {
		db *gorm.DB
	}
)

func NewUsers(db *gorm.DB) Users {
	return Users{
		db,
	}
}

// Create a new user
func (u Users) Create(user *models.User) {
	u.db.Create(user)
}

// Update a user
func (u Users) Update(user *models.User) error {
	dbUser, found := u.UserWithID(user.ID)
	if !found {
		return repo.ErrModelNotFound
	}

	u.db.Model(&dbUser).Updates(user).RecordNotFound()

	return nil
}

// UserWithID returns a User with id
func (u Users) UserWithID(id string) (user models.User, found bool) {
	found = !u.db.First(&user, "id = ?", id).RecordNotFound()
	return
}

// Delete a user
func (u Users) Delete(id string) error {
	user, found := u.UserWithID(id)
	if !found {
		return repo.ErrModelNotFound
	}

	u.db.Delete(user)

	return nil
}

// List all users
func (u Users) List(page models.Page) (users []models.User, next string) {
	query := u.db.Limit(page.Count + 1)

	if page.ContinuationID != "" {
		user, found := u.UserWithID(page.ContinuationID)
		if found {
			query = query.Where("created_at >= ?", user.CreatedAt)
		}
	}

	query.Find(&users)

	if len(users) > page.Count {
		next = users[len(users)-1].ID
		users = users[:len(users)-1]
	}

	return
}

// UserWithName returns a User with username
func (u Users) UserWithName(name string) (user models.User, found bool) {
	found = !u.db.First(&user, "username = ?", name).RecordNotFound()
	return
}
