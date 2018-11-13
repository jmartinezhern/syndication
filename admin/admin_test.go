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
	"github.com/varddum/syndication/database"
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

	s.client, err = rpc.Dial("unixpacket", TestSocketPath)
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
	database.NewUser("test", "testtesttest")

	args := NewUserArgs{
		"test",
		"testtesttest",
	}

	var msg string
	err := s.client.Call("Admin.NewUser", args, &msg)
	s.EqualError(err, "Username already exists")
}

func (s *AdminTestSuite) TestDeleteUser() {
	user := database.NewUser("test", "testtesttest")

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
	user1 := database.NewUser("test1", "testtesttest")
	user2 := database.NewUser("test2", "testtesttest")

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
	user := database.NewUser("test", "testtesttest")

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
	user := database.NewUser("test", "testtesttest")

	var msg string
	args := ChangeUserPasswordArgs{
		UserID:      user.APIID,
		NewPassword: "gopherpass",
	}

	err := s.client.Call("Admin.ChangeUserPassword", args, &msg)
	s.Nil(err)

	_, ok := database.UserWithCredentials(user.Username, "gopherpass")
	s.True(ok)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
