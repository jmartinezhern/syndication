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

package admin

import (
	"net/rpc"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	AdminTestSuite struct {
		suite.Suite

		serv *Service

		client *rpc.Client
	}
)

const (
	TestDBPath     = "/tmp/syndication-test-admin.db"
	TestSocketPath = "/tmp/syndication.socket"
)

func (s *AdminTestSuite) SetupTest() {
	err := database.Init("sqlite3", TestDBPath)
	s.Require().Nil(err)

	s.serv, err = NewService(TestSocketPath)
	s.Require().NotNil(s.serv)
	s.Require().Nil(err)

	s.serv.Start()

	s.client, err = rpc.Dial("unix", TestSocketPath)
	s.Require().Nil(err)
}

func (s *AdminTestSuite) TearDownTest() {
	s.serv.Stop()

	err := database.Close()
	s.Nil(err)

	err = os.Remove(TestDBPath)
	s.Nil(err)

	s.client.Close()
}

func (s *AdminTestSuite) TestNewUser() {
	args := NewUserArgs{
		"test",
		"testtesttest",
	}
	var msg string
	err := s.client.Call("Admin.NewUser", args, &msg)
	s.Nil(err)

	_, found := database.UserWithName("test")
	s.True(found)
}

func (s *AdminTestSuite) TestNewConflictingUser() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")

	user := models.User{
		Username:     "test",
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	database.CreateUser(&user)

	args := NewUserArgs{
		"test",
		"testtesttest",
	}

	var msg string
	err := s.client.Call("Admin.NewUser", args, &msg)
	s.EqualError(err, "Username already exists")
}

func (s *AdminTestSuite) TestDeleteUser() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")

	user := models.User{
		Username:     "test",
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	database.CreateUser(&user)

	var msg string
	err := s.client.Call("Admin.DeleteUser", user.APIID, &msg)
	s.Nil(err)

	_, found := database.UserWithAPIID(user.APIID)
	s.False(found)
}

func (s *AdminTestSuite) TestGetNonExistentUserID() {
	var userID string
	err := s.client.Call("Admin.GetUserID", "test", &userID)
	s.NotNil(err)
	s.Empty(userID)
}

func (s *AdminTestSuite) TestGetUsers() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")

	user1 := models.User{
		Username:     "test1",
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	user2 := models.User{
		Username:     "test2",
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	database.CreateUser(&user1)
	database.CreateUser(&user2)

	var users []User

	err := s.client.Call("Admin.GetUsers", 2, &users)
	s.Nil(err)
	s.Len(users, 2)

	s.NotZero(sort.Search(len(users), func(i int) bool {
		return users[i].Name == user1.Username && users[i].ID == user1.APIID
	}))
	s.NotZero(sort.Search(len(users), func(i int) bool {
		return users[i].Name == user2.Username && users[i].ID == user2.APIID
	}))
}

func (s *AdminTestSuite) TestChangeUserName() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")

	user := models.User{
		Username:     "test",
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	database.CreateUser(&user)

	var msg string
	args := ChangeUserNameArgs{
		UserID:  user.APIID,
		NewName: "gopher",
	}

	err := s.client.Call("Admin.ChangeUserName", args, &msg)
	s.Nil(err)

	modifiedUser, found := database.UserWithAPIID(user.APIID)
	s.Require().True(found)

	s.Equal("gopher", modifiedUser.Username)
}

func (s *AdminTestSuite) TestChangeUserPassword() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")

	user := models.User{
		Username:     "test",
		PasswordHash: hash,
		PasswordSalt: salt,
	}

	database.CreateUser(&user)

	var msg string
	args := ChangeUserPasswordArgs{
		UserID:      user.APIID,
		NewPassword: "gopherpass",
	}

	err := s.client.Call("Admin.ChangeUserPassword", args, &msg)
	s.Nil(err)

	user, _ = database.UserWithName("test")
	s.True(utils.VerifyPasswordHash("gopherpass", user.PasswordHash, user.PasswordSalt))
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
