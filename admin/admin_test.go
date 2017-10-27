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
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/varddum/syndication/config"
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

func (suite *AdminTestSuite) SetupTest() {
	var err error
	suite.db, err = database.NewDB(config.Database{
		Type:             "sqlite3",
		Connection:       TestDBPath,
		APIKeyExpiration: config.Duration{Duration: time.Hour * 72},
	})
	suite.Nil(err)

	suite.socketPath = "/tmp/syndication.socket"
	suite.admin, err = NewAdmin(suite.db, suite.socketPath)
	suite.Require().NotNil(suite.admin)
	suite.Require().Nil(err)

	suite.admin.Start()

	suite.conn, err = net.Dial("unixpacket", suite.socketPath)
	suite.Require().Nil(err)
}

func (suite *AdminTestSuite) TearDownTest() {
	stopLock := sync.Mutex{}
	stopLock.Lock()
	suite.admin.Stop(true)
	stopLock.Unlock()
	err := suite.db.Close()
	suite.Nil(err)

	err = os.Remove(TestDBPath)
	suite.Nil(err)

	err = suite.conn.Close()
	suite.Nil(err)
}

func (suite *AdminTestSuite) TestBadCommandArgument() {
	message := `
	{
		"command": "bogus"
	}
	`
	size, err := suite.conn.Write([]byte(message))
	suite.Require().Nil(err)
	suite.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	result := &Response{}
	err = json.Unmarshal(buff[:size], &result)
	suite.Require().Nil(err)
	suite.Equal(NotImplemented, result.Status)
	suite.Equal("bogus is not implemented.", result.Error)
}

func (suite *AdminTestSuite) TestNewUser() {
	message := `
	{
		"command": "NewUser",
		"arguments": {"username":"GoTest",
							    "password":"testtesttest"}
	}
	`
	size, err := suite.conn.Write([]byte(message))
	suite.Require().Nil(err)
	suite.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	users := suite.db.Users("username")
	suite.Require().Len(users, 1)

	suite.Equal(users[0].Username, "GoTest")
	suite.NotEmpty(users[0].ID)
	suite.NotEmpty(users[0].APIID)
}

func (suite *AdminTestSuite) TestNewUserWithBadFirstArgument() {
	message := `
	{
		"command": "NewUser",
		"arguments": {"bogus":"GoTest",
							    "password":"testtesttest"}
	}
	`
	size, err := suite.conn.Write([]byte(message))
	suite.Require().Nil(err)
	suite.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	type UsersResult struct {
		Status StatusCode    `json:"status"`
		Error  string        `json:"Error"`
		Result []models.User `json:"result"`
	}

	var result UsersResult
	err = json.Unmarshal(buff[:size], &result)
	suite.Require().Nil(err)
	suite.Equal("Bad first argument", result.Error)

	users := suite.db.Users("username")
	suite.Require().Len(users, 0)
}

func (suite *AdminTestSuite) TestNewUserWithBadSecondArgument() {
	message := `
	{
		"command": "NewUser",
		"arguments": {"username":"GoTest",
							    "bogus":"testtesttest"}
	}
	`
	size, err := suite.conn.Write([]byte(message))
	suite.Require().Nil(err)
	suite.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	type UsersResult struct {
		Status StatusCode    `json:"status"`
		Error  string        `json:"Error"`
		Result []models.User `json:"result"`
	}

	var result UsersResult
	err = json.Unmarshal(buff[:size], &result)
	suite.Require().Nil(err)
	suite.Equal("Bad second argument", result.Error)

	users := suite.db.Users("username")
	suite.Require().Len(users, 0)
}

func (suite *AdminTestSuite) TestGetUsers() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	message := `{
		"command": "GetUsers"
	}
	`

	size, err := suite.conn.Write([]byte(message))
	suite.Require().Nil(err)
	suite.Equal(len(message), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	buff = buff[:size]

	type UsersResult struct {
		Status StatusCode    `json:"status"`
		Error  string        `json:"Error"`
		Result []models.User `json:"result"`
	}

	result := &UsersResult{}
	err = json.Unmarshal(buff, result)
	suite.Require().Nil(err)
	suite.Require().Equal(OK, result.Status)
	suite.Require().Len(result.Result, 1)
	suite.NotEmpty(result.Result[0].APIID)
}

func (suite *AdminTestSuite) TestGetUser() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	cmd := Request{
		Command: "GetUser",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
		},
	}

	b, err := json.Marshal(cmd)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	buff = buff[:size]

	type UsersResult struct {
		Status  int         `json:"status"`
		Message string      `json:"message"`
		Result  models.User `json:"result"`
	}

	result := &UsersResult{}
	err = json.Unmarshal(buff, result)

	suite.Require().Nil(err)
	suite.NotEmpty(result.Result.APIID)
}

func (suite *AdminTestSuite) TestGetUserWithBadArgument() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	cmd := Request{
		Command: "GetUser",
		Arguments: map[string]interface{}{
			"bogus": user.APIID,
		},
	}

	b, err := json.Marshal(cmd)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	result := &Response{}
	err = json.Unmarshal(buff[:size], result)
	suite.Require().Nil(err)

	suite.Equal(BadArgument, result.Status)
	suite.Equal("Bad first argument", result.Error)
}

func (suite *AdminTestSuite) TestDeleteUser() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	users := suite.db.Users()
	suite.Len(users, 1)

	user := users[0]

	cmd := Request{
		Command: "DeleteUser",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
		},
	}

	b, err := json.Marshal(cmd)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	buff = buff[:size]

	type UsersResult struct {
		Status  StatusCode  `json:"status"`
		Message string      `json:"message"`
		Result  models.User `json:"result"`
	}

	result := &UsersResult{}
	err = json.Unmarshal(buff, result)

	suite.Require().Nil(err)
	suite.Equal(OK, result.Status)

	users = suite.db.Users()
	suite.Len(users, 0)
}

func (suite *AdminTestSuite) TestDeleteUserWithBadArgument() {
	cmd := Request{
		Command: "DeleteUser",
		Arguments: map[string]interface{}{
			"bogus": "random",
		},
	}

	b, err := json.Marshal(cmd)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	result := &Response{}
	err = json.Unmarshal(buff[:size], result)

	suite.Require().Nil(err)
	suite.Equal(BadArgument, result.Status)
	suite.Equal("Bad first argument", result.Error)
}

func (suite *AdminTestSuite) TestChangeUserName() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"userID":  user.APIID,
			"newName": "gopher",
		},
	}

	b, err := json.Marshal(req)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	buff = buff[:size]

	resp := &Response{}
	err = json.Unmarshal(buff, resp)
	suite.Require().Nil(err)
	suite.Equal(OK, resp.Status)

	user, err = suite.db.UserWithAPIID(user.APIID)
	suite.Equal("gopher", user.Username)
}

func (suite *AdminTestSuite) TestChangeUserNameWithBadFirstArgument() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"bogus":   user.APIID,
			"newName": "gopher",
		},
	}

	b, err := json.Marshal(req)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	suite.Require().Nil(err)
	suite.Equal(BadArgument, resp.Status)
	suite.Equal("Bad first argument", resp.Error)
}

func (suite *AdminTestSuite) TestChangeUserNameWithBadSecondArgument() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
			"bogus":  "gopher",
		},
	}

	b, err := json.Marshal(req)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	suite.Require().Nil(err)
	suite.Equal(BadArgument, resp.Status)
	suite.Equal("Bad second argument", resp.Error)
}

func (suite *AdminTestSuite) TestChangeUserPassword() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)
	suite.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"userID":      user.APIID,
			"newPassword": "gopher",
		},
	}

	b, err := json.Marshal(req)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	buff = buff[:size]

	resp := &Response{}
	err = json.Unmarshal(buff, resp)
	suite.Require().Nil(err)
	suite.Equal(OK, resp.Status)

	user, err = suite.db.Authenticate("GoTest", "gopher")
	suite.Nil(err)
	suite.NotEmpty(user.APIID)
}

func (suite *AdminTestSuite) TestChangeUserPasswordFirstArgument() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)
	suite.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"bogus":       user.APIID,
			"newPassword": "gopher",
		},
	}

	b, err := json.Marshal(req)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	suite.Require().Nil(err)
	suite.Equal(BadArgument, resp.Status)
	suite.Equal("Bad first argument", resp.Error)
}

func (suite *AdminTestSuite) TestChangeUserPasswordSecondArgument() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)
	suite.Require().NotEmpty(user.APIID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"userID": user.APIID,
			"bogus":  "gopher",
		},
	}

	b, err := json.Marshal(req)
	size, err := suite.conn.Write(b)
	suite.Require().Nil(err)
	suite.Equal(len(b), size)

	buff := make([]byte, 512)
	size, err = suite.conn.Read(buff)
	suite.Require().Nil(err)

	resp := &Response{}
	err = json.Unmarshal(buff[:size], resp)
	suite.Require().Nil(err)
	suite.Equal(BadArgument, resp.Status)
	suite.Equal("Bad second argument", resp.Error)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
