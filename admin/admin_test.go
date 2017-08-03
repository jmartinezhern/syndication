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
	"testing"

	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/models"
	"github.com/stretchr/testify/suite"
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
	suite.db, err = database.NewDB("sqlite3", TestDBPath)
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
	err := suite.conn.Close()
	suite.Nil(err)
	suite.admin.Stop(true)
	err = suite.db.Close()
	suite.Nil(err)

	err = os.Remove(suite.db.Connection)
	suite.Nil(err)
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
	suite.NotEmpty(users[0].UUID)
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
	suite.NotEmpty(result.Result[0].UUID)
}

func (suite *AdminTestSuite) TestGetUser() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	cmd := Request{
		Command: "GetUser",
		Arguments: map[string]interface{}{
			"userID": user.UUID,
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
	suite.NotEmpty(result.Result.UUID)
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
			"userID": user.UUID,
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

func (suite *AdminTestSuite) TestChangeUserName() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	req := Request{
		Command: "ChangeUserName",
		Arguments: map[string]interface{}{
			"userID":  user.UUID,
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

	user, err = suite.db.UserWithUUID(user.UUID)
	suite.Equal("gopher", user.Username)
}

func (suite *AdminTestSuite) TestChangeUserPassword() {
	err := suite.db.NewUser("GoTest", "testtesttest")
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)
	suite.Require().NotEmpty(user.UUID)

	req := Request{
		Command: "ChangeUserPassword",
		Arguments: map[string]interface{}{
			"userID":      user.UUID,
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
	suite.NotEmpty(user.UUID)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
