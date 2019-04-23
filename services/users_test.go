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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	UsersSuite struct {
		suite.Suite

		db      *sql.DB
		service Users
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
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	})

	_, err := s.service.NewUser("gopher", "password")
	s.EqualError(err, ErrUsernameConflicts.Error())
}

func (s *UsersSuite) TestDeleteUser() {
	userID := utils.CreateAPIID()
	s.repo.Create(&models.User{
		APIID:    userID,
		Username: "gopher",
	})

	s.NoError(s.service.DeleteUser(userID))
}

func (s *UsersSuite) TestUser() {
	userID := utils.CreateAPIID()
	s.repo.Create(&models.User{
		APIID:    userID,
		Username: "gopher",
	})

	user, found := s.service.User(userID)
	s.True(found)

	s.Equal("gopher", user.Username)
}

func (s *UsersSuite) SetupTest() {
	s.db = sql.NewDB("sqlite3", ":memory:")
	s.repo = sql.NewUsers(s.db)

	s.service = NewUsersService(s.repo)
}

func (s *UsersSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
