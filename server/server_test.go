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

package server

import (
	"github.com/labstack/echo"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
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

		server *Server
		user   models.User
		rec    *httptest.ResponseRecorder
		e      *echo.Echo
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (t *ServerTestSuite) SetupTest() {
	err := database.Init("sqlite3", testDBPath)
	t.Require().NoError(err)

	t.server = NewServer("secret_cat")
	t.server.handle.Logger.SetLevel(log.OFF)
	t.server.handle.HideBanner = true

	randUserName := RandStringRunes(8)
	t.user = database.NewUser("123456", randUserName)

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
