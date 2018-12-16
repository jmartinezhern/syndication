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

package test

import (
	"github.com/labstack/echo"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/server"
)

const (
	testDBPath = "/tmp/syndication-test-server.db"

	testHost = "localhost"

	testHTTPPort = 9876
)

var (
	testBaseURL = "http://" + testHost + ":" + strconv.Itoa(testHTTPPort)
)

var mockRSSServer *httptest.Server

type (
	ServerTestSuite struct {
		suite.Suite

		server   *server.Server
		user     models.User
		rec      *httptest.ResponseRecorder
		e        *echo.Echo
		unctgCtg models.Category
	}
)

func (t *ServerTestSuite) SetupTest() {
	err := database.Init("sqlite3", testDBPath)
	t.Require().NoError(err)

	t.server = server.NewServer("secret_cat")

	randUserName := RandStringRunes(8)
	t.user = database.NewUser("123456", randUserName)

	t.unctgCtg = database.NewCategory(models.Uncategorized, t.user)

	t.rec = httptest.NewRecorder()
	t.e = echo.New()
}

func (t *ServerTestSuite) TearDownTest() {
	os.Remove(testDBPath)
}

func TestServerTestSuite(t *testing.T) {
	dir := http.Dir(os.Getenv("GOPATH") + "/src/github.com/jmartinezhern/syndication/server/")
	mockRSSServer = httptest.NewServer(http.FileServer(dir))
	defer mockRSSServer.Close()

	suite.Run(t, new(ServerTestSuite))
}
