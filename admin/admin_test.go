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
	"encoding/json"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

type (
	AdminTestSuite struct {
		suite.Suite

		db         *database.DB
		admin      *Admin
		conn       net.Conn
		socketPath string
	}
)

const TestDBPath = "/tmp/syndication-test-admin.db"

func (s *AdminTestSuite) SetupTest() {
	var err error
	s.db, err = database.NewDB("sqlite3", TestDBPath)
	s.Nil(err)

	s.socketPath = "/tmp/syndication.socket"
	s.admin, err = NewAdmin(s.db, s.socketPath)
	s.Require().NotNil(s.admin)
	s.Require().Nil(err)

	s.admin.Start()

	s.conn, err = net.Dial("unixpacket", s.socketPath)
	s.Require().Nil(err)
}

func (s *AdminTestSuite) TearDownTest() {
	stopLock := sync.Mutex{}
	stopLock.Lock()
	s.admin.Stop(true)
	stopLock.Unlock()
	err := s.db.Close()
	s.Nil(err)

	err = os.Remove(TestDBPath)
	s.Nil(err)

	err = s.conn.Close()
	s.Nil(err)
}

func (s *AdminTestSuite) TestBadCommandArgument() {
	message := `
	{
		"command": "bogus"
	}
	`
	size, err := s.conn.Write([]byte(message))
	s.Require().Nil(err)
	s.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	result := &Response{}
	err = json.Unmarshal(buff[:size], &result)
	s.Require().Nil(err)
	s.Equal(NotImplemented, result.Status)
	s.Equal("bogus is not implemented.", result.Error)
}

func (s *AdminTestSuite) TestNewUser() {
	message := `
	{
		"command": "NewUser",
		"arguments": {"username":"GoTest",
							    "password":"testtesttest"}
	}
	`
	size, err := s.conn.Write([]byte(message))
	s.Require().Nil(err)
	s.Equal(len(message), size)

	buff := make([]byte, 256)
	_, err = s.conn.Read(buff)
	s.Require().Nil(err)

	users := s.db.Users("username")
	s.Require().Len(users, 1)

	s.Equal(users[0].Username, "GoTest")
	s.NotEmpty(users[0].ID)
	s.NotEmpty(users[0].APIID)
}

func (s *AdminTestSuite) TestNewUserWithBadFirstArgument() {
	message := `
	{
		"command": "NewUser",
		"arguments": {"bogus":"GoTest",
							    "password":"testtesttest"}
	}
	`
	size, err := s.conn.Write([]byte(message))
	s.Require().Nil(err)
	s.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	type UsersResult struct {
		Status StatusCode    `json:"status"`
		Error  string        `json:"Error"`
		Result []models.User `json:"result"`
	}

	var result UsersResult
	err = json.Unmarshal(buff[:size], &result)
	s.Require().Nil(err)
	s.Equal("Bad first argument", result.Error)

	users := s.db.Users("username")
	s.Require().Len(users, 0)
}

func (s *AdminTestSuite) TestNewUserWithBadSecondArgument() {
	message := `
	{
		"command": "NewUser",
		"arguments": {"username":"GoTest",
							    "bogus":"testtesttest"}
	}
	`
	size, err := s.conn.Write([]byte(message))
	s.Require().Nil(err)
	s.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	type UsersResult struct {
		Status StatusCode    `json:"status"`
		Error  string        `json:"Error"`
		Result []models.User `json:"result"`
	}

	var result UsersResult
	err = json.Unmarshal(buff[:size], &result)
	s.Require().Nil(err)
	s.Equal("Bad second argument", result.Error)

	users := s.db.Users("username")
	s.Require().Len(users, 0)
}

func (s *AdminTestSuite) TestGetUsers() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	message := `{
		"command": "GetUsers"
	}
	`

	size, err := s.conn.Write([]byte(message))
	s.Require().Nil(err)
	s.Equal(len(message), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	buff = buff[:size]

	type UsersResult struct {
		Status StatusCode    `json:"status"`
		Error  string        `json:"Error"`
		Result []models.User `json:"result"`
	}

	result := &UsersResult{}
	err = json.Unmarshal(buff, result)
	s.Require().Nil(err)
	s.Require().Equal(OK, result.Status)
	s.Require().Len(result.Result, 1)
	s.NotEmpty(result.Result[0].APIID)
}

func (s *AdminTestSuite) TestGetUser() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	user, found := s.db.UserWithName("GoTest")
	s.Require().True(found)

	cmd := Request{
		Command: "GetUser",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
		},
	}

	b, err := json.Marshal(cmd)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	buff = buff[:size]

	type UsersResult struct {
		Status  int         `json:"status"`
		Message string      `json:"message"`
		Result  models.User `json:"result"`
	}

	result := &UsersResult{}
	err = json.Unmarshal(buff, result)

	s.Require().Nil(err)
	s.NotEmpty(result.Result.APIID)
}

func (s *AdminTestSuite) TestGetUserWithBadArgument() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	cmd := Request{
		Command: "GetUser",
		Arguments: map[string]interface{}{
			"bogus": user.APIID,
		},
	}

	b, err := json.Marshal(cmd)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	result := &Response{}
	err = json.Unmarshal(buff[:size], result)
	s.Require().Nil(err)

	s.Equal(BadArgument, result.Status)
	s.Equal("Bad first argument", result.Error)
}

func (s *AdminTestSuite) TestDeleteUser() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	users := s.db.Users()
	s.Len(users, 1)

	cmd := Request{
		Command: "DeleteUser",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
		},
	}

	b, err := json.Marshal(cmd)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	buff = buff[:size]

	type UsersResult struct {
		Status  StatusCode  `json:"status"`
		Message string      `json:"message"`
		Result  models.User `json:"result"`
	}

	result := &UsersResult{}
	err = json.Unmarshal(buff, result)

	s.Require().Nil(err)
	s.Equal(OK, result.Status)

	users = s.db.Users()
	s.Len(users, 0)
}

func (s *AdminTestSuite) TestDeleteUserWithBadArgument() {
	cmd := Request{
		Command: "DeleteUser",
		Arguments: map[string]interface{}{
			"bogus": "random",
		},
	}

	b, err := json.Marshal(cmd)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 256)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	result := &Response{}
	err = json.Unmarshal(buff[:size], result)

	s.Require().Nil(err)
	s.Equal(BadArgument, result.Status)
	s.Equal(badArgumentErrorStr(1), result.Error)
}

func (s *AdminTestSuite) TestChangeUserName() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"userID":  user.APIID,
			"newName": "gopher",
		},
	}

	b, err := json.Marshal(req)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	buff = buff[:size]

	resp := &Response{}
	err = json.Unmarshal(buff, resp)
	s.Require().Nil(err)
	s.Equal(OK, resp.Status)

	user, found := s.db.UserWithAPIID(user.APIID)
	s.True(found)
	s.Equal("gopher", user.Username)
}

func (s *AdminTestSuite) TestChangeUserNameWithBadFirstArgument() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"bogus":   user.APIID,
			"newName": "gopher",
		},
	}

	b, err := json.Marshal(req)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 256)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	s.Require().Nil(err)
	s.Equal(BadArgument, resp.Status)
	s.Equal(badArgumentErrorStr(1), resp.Error)
}

func (s *AdminTestSuite) TestChangeUserNameWithBadSecondArgument() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
			"bogus":  "gopher",
		},
	}

	b, err := json.Marshal(req)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 256)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	s.Require().Nil(err)
	s.Equal(BadArgument, resp.Status)
	s.Equal(badArgumentErrorStr(2), resp.Error)
}

func (s *AdminTestSuite) TestChangeUserPassword() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"userID":      user.APIID,
			"newPassword": "gopher",
		},
	}

	b, err := json.Marshal(req)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	buff = buff[:size]

	resp := &Response{}
	err = json.Unmarshal(buff, resp)
	s.Require().Nil(err)
	s.Equal(OK, resp.Status)

	user, found := s.db.UserWithCredentials("GoTest", "gopher")
	s.True(found)
	s.NotEmpty(user.APIID)
}

func (s *AdminTestSuite) TestChangeUserPasswordFirstArgument() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"bogus":       user.APIID,
			"newPassword": "gopher",
		},
	}

	b, err := json.Marshal(req)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	s.Require().Nil(err)
	s.Equal(BadArgument, resp.Status)
	s.Equal(badArgumentErrorStr(1), resp.Error)
}

func (s *AdminTestSuite) TestChangeUserPasswordSecondArgument() {
	user := s.db.NewUser("GoTest", "testtesttest")
	s.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
			"bogus":  "gopher",
		},
	}

	b, err := json.Marshal(req)
	s.Require().Nil(err)

	size, err := s.conn.Write(b)
	s.Require().Nil(err)
	s.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = s.conn.Read(buff)
	s.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	s.Require().Nil(err)
	s.Equal(BadArgument, resp.Status)
	s.Equal(badArgumentErrorStr(2), resp.Error)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
