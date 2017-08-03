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
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

var mockRSSServer *httptest.Server

type (
	ServerTestSuite struct {
		suite.Suite

		server   *Server
		user     models.User
		rec      *httptest.ResponseRecorder
		e        *echo.Echo
		unctgCtg models.Category
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (s *ServerTestSuite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<rss></rss>")
}

func (s *ServerTestSuite) SetupTest() {
	err := database.Init("sqlite3", ":memory:")
	s.Require().NoError(err)

	s.server = NewServer("secret_cat")
	s.server.handle.Logger.SetLevel(log.OFF)
	s.server.handle.HideBanner = true

	randUserName := randStringRunes(8)

	_, err = s.server.auth.Register(randUserName, "1234546")
	s.Require().NoError(err)

	var found bool
	s.user, found = database.UserWithName(randUserName)
	s.Require().True(found)

	s.unctgCtg, found = database.CategoryWithName(models.Uncategorized, s.user)
	s.Require().True(found)

	s.rec = httptest.NewRecorder()
	s.e = echo.New()
}

func (s *ServerTestSuite) TearDownTest() {
	err := database.Close()
	s.NoError(err)
}

func TestServerTestSuite(t *testing.T) {
	s := &ServerTestSuite{}
	server := &http.Server{
		Addr:    ":9090",
		Handler: s,
	}

	go server.ListenAndServe()

	time.Sleep(time.Second)

	defer server.Close()

	suite.Run(t, s)
}
