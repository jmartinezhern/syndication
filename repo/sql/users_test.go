/*
  Copyright (C) 2017 Jorge Martinez Hernandez

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more detailt.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package sql

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type UsersSuite struct {
	suite.Suite

	db   *DB
	repo repo.Users
}

func (s *UsersSuite) TestCreate() {
	user := models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}

	s.repo.Create(&user)

	fUser, found := s.repo.UserWithName("gopher")
	s.True(found)
	s.Equal(user.APIID, fUser.APIID)
	s.Equal(user.Username, fUser.Username)
}

func (s *UsersSuite) TestList() {
	s.repo.Create(&models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test_one",
	})
	s.repo.Create(&models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test_two",
	})

	users, next := s.repo.List("", 1)
	s.NotEmpty(next)
	s.Require().Len(users, 1)
	s.Equal("test_one", users[0].Username)

	users, _ = s.repo.List(next, 1)
	s.Require().Len(users, 1)
	s.Equal(users[0].APIID, next)
	s.Equal("test_two", users[0].Username)

}

func (s *UsersSuite) TestUserWithID() {
	userID := utils.CreateAPIID()
	s.repo.Create(&models.User{
		APIID:    userID,
		Username: "test",
	})

	user, found := s.repo.UserWithID(userID)
	s.True(found)
	s.Equal(userID, user.APIID)
	s.Equal("test", user.Username)
}

func (s *UsersSuite) TestDelete() {
	user := models.User{
		Username: "test",
		APIID:    utils.CreateAPIID(),
	}
	s.repo.Create(&user)

	s.NoError(s.repo.Delete(user.APIID))

	_, found := s.repo.UserWithID(user.APIID)
	s.False(found)
}

func (s *UsersSuite) TestDeleteUnknownUser() {
	s.EqualError(s.repo.Delete("bogus"), repo.ErrModelNotFound.Error())
}

func (s *UsersSuite) TestUserWithName() {
	s.repo.Create(&models.User{
		Username: "gopher",
	})

	user, found := s.repo.UserWithName("gopher")
	s.True(found)
	s.Equal("gopher", user.Username)
}

func (s *UsersSuite) TestUserWithUnknownName() {
	_, found := s.repo.UserWithName("bogus")
	s.False(found)
}

func (s *UsersSuite) SetupTest() {
	s.db = NewDB("sqlite3", ":memory:")

	s.repo = NewUsers(s.db)
}

func (s *UsersSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
