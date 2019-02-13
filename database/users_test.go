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

package database

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type UsersSuite struct {
	suite.Suite
}

// This tests take into account the user created in Test Setup

func (t *UsersSuite) TestCreateUser() {
	user := models.User{
		Username: "test",
	}

	CreateUser(&user)

	user, found := UserWithName("test")
	t.True(found)
	t.NotZero(user.ID)
}

func (t *UsersSuite) TestUsers() {
	CreateUser(&models.User{Username: "test_one"})
	CreateUser(&models.User{Username: "test_two"})

	users := Users()
	t.Len(users, 2)
}

func (t *UsersSuite) TestUserWithAPIID() {
	userID := utils.CreateAPIID()

	CreateUser(&models.User{
		APIID:    userID,
		Username: "test",
	})

	user, found := UserWithAPIID(userID)
	t.True(found)
	t.Equal(userID, user.APIID)
	t.Equal("test", user.Username)
}

func (t *UsersSuite) TestDeleteUser() {
	user := models.User{
		Username: "test",
		APIID:    utils.CreateAPIID(),
	}
	CreateUser(&user)

	t.NoError(DeleteUser(user.APIID))

	_, found := UserWithAPIID(user.APIID)
	t.False(found)
}

func (t *UsersSuite) TestDeleteUnknownUser() {
	t.EqualError(DeleteUser("bogus"), ErrModelNotFound.Error())
}

func (t *UsersSuite) TestUserWithUnknownAPIID() {
	_, found := UserWithAPIID("bogus")
	t.False(found)
}

func (t *UsersSuite) TestUserWithName() {
	CreateUser(&models.User{
		Username: "gopher",
	})

	user, found := UserWithName("gopher")
	t.True(found)
	t.Equal("gopher", user.Username)
}

func (t *UsersSuite) TestUserWithUnknownName() {
	_, found := UserWithName("bogus")
	t.False(found)
}

func (t *UsersSuite) SetupTest() {
	err := Init("sqlite3", ":memory:")
	t.Require().NoError(err)
}

func (t UsersSuite) TearDownTest() {
	err := Close()
	t.Require().NoError(err)
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
