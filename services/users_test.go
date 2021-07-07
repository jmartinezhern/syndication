/*
 *   Copyright (C) 2021. Jorge Martinez Hernandez
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU Affero General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU Affero General Public License for more details.
 *
 *   You should have received a copy of the GNU Affero General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package services_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	UsersSuite struct {
		suite.Suite

		db      *gorm.DB
		service services.Users
		repo    repo.Users
	}
)

func (s *UsersSuite) TestNewUser() {
	_, err := s.service.NewUser("gopher", "passw0rd!")
	s.NoError(err)

	user, found := s.repo.UserWithName("gopher")
	s.True(found)
	s.Equal("gopher", user.Username)
}

func (s *UsersSuite) TestNewConflictingUser() {
	s.repo.Create(&models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	})

	_, err := s.service.NewUser("gopher", "password")
	s.EqualError(err, services.ErrUsernameConflicts.Error())
}

func (s *UsersSuite) TestDeleteUser() {
	userID := utils.CreateID()

	s.repo.Create(&models.User{
		ID:       userID,
		Username: "gopher",
	})

	s.NoError(s.service.DeleteUser(userID))
}

func (s *UsersSuite) TestDeleteMissingUser() {
	s.EqualError(s.service.DeleteUser("bogus"), services.ErrUserNotFound.Error())
}

func (s *UsersSuite) TestUser() {
	userID := utils.CreateID()

	s.repo.Create(&models.User{
		ID:       userID,
		Username: "gopher",
	})

	user, found := s.service.User(userID)
	s.True(found)

	s.Equal("gopher", user.Username)
}

func (s *UsersSuite) SetupTest() {
	var err error

	s.db, err = gorm.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	sql.AutoMigrateTables(s.db)

	s.repo = sql.NewUsers(s.db)

	s.service = services.NewUsersService(s.repo)
}

func (s *UsersSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
