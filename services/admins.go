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

type (
	// Admins interface defines the Users services
	Admins interface {
		// CreateInitialAdmin creates a the owner or initial admin with user name and password
		CreateInitialAdmin(username, password string)

		// NewAdmin creates a new admin with username and password
		NewAdmin(username, password string) (models.Admin, error)

		// Update an admin
		ChangePassword(adminID, newPassword string) error
	}

	AdminsService struct {
		adminsRepo repo.Admins
	}
)

var (
	// ErrAdminNotFound signals that admin was not found
	ErrAdminNotFound = errors.New("admin not found")
)

func NewAdminsService(adminsRepo repo.Admins) AdminsService {
	return AdminsService{
		adminsRepo,
	}
}

func (a AdminsService) NewAdmin(username, password string) (models.Admin, error) {
	if _, found := a.adminsRepo.AdminWithName(username); found {
		return models.Admin{}, ErrUsernameConflicts
	}

	hash, salt := utils.CreatePasswordHashAndSalt(password)

	admin := models.Admin{
		ID:           utils.CreateID(),
		Username:     username,
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	a.adminsRepo.Create(&admin)

	return admin, nil
}

func (a AdminsService) ChangePassword(adminID, newPassword string) error {
	admin, found := a.adminsRepo.AdminWithID(adminID)
	if !found {
		return ErrAdminNotFound
	}

	hash, salt := utils.CreatePasswordHashAndSalt(newPassword)
	admin.PasswordHash = hash
	admin.PasswordSalt = salt

	return a.adminsRepo.Update(adminID, &admin)
}
