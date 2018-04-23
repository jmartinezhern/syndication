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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

const (
	TestDBPath = "/tmp/syndication-test-server.db"

	testHTTPPort = 9876
)

var (
	testBaseURL = "http://localhost:" + strconv.Itoa(testHTTPPort)
)

var mockRSSServer *httptest.Server

type (
	ServerTestSuite struct {
		suite.Suite

		gDB    *database.DB
		db     database.UserDB
		server *Server
		user   models.User
		token  string
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

func (s *ServerTestSuite) SetupTest() {
	var err error
	s.gDB, err = database.NewDB("sqlite3", TestDBPath)
	s.Require().Nil(err)

	s.server = NewServer(s.gDB)
	s.server.handle.HideBanner = true
	go s.server.Start("localhost", 9876)

	randUserName := RandStringRunes(8)

	user := s.gDB.NewUser(randUserName, "testtesttest")
	s.db = s.gDB.NewUserDB(user)

	s.Require().NotEmpty(user.APIID)

	token, err := s.db.NewAPIKey(s.server.authSecret, time.Hour*72)
	s.Require().Nil(err)
	s.Require().NotEmpty(token.Key)

	s.token = token.Key
	s.user = user
}

func (s *ServerTestSuite) TearDownTest() {
	os.Remove(TestDBPath)

	s.server.Stop()
}

func (s *ServerTestSuite) TestGetStats() {
	feed := s.db.NewFeed("News", "http://example.com")
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.MarkerRead,
			Saved: true,
		}

		s.db.NewEntry(entry, feed.APIID)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.MarkerUnread,
		}

		s.db.NewEntry(entry, feed.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries/stats", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	s.Require().Nil(err)

	s.Equal(7, respStats.Unread)
	s.Equal(3, respStats.Read)
	s.Equal(3, respStats.Saved)
	s.Equal(10, respStats.Total)
}

func (s *ServerTestSuite) TestOPMLImport() {
	data := []byte(`
	<opml>
		<body>
			<outline text="Sports" title="Sports">
				<outline type="rss"  text="Basketball" title="Basketball" xmlUrl="http://example.com/basketball" htmlUrl="http://example.com/basketball"/>
			</outline>
			<outline type="rss" text="Baseball" title="Baseball" xmlUrl="http://example.com/baseball" htmlUrl="http://example.com/baseball"/>
		</body>
	</opml>
	`)

	req, err := http.NewRequest("POST", "http://localhost:9876/v1/import", bytes.NewBuffer(data))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/xml")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	ctgs := s.db.Categories()
	s.Require().Len(ctgs, 2)

	sportsCtg, ok := s.db.CategoryWithName("Sports")
	s.Require().True(ok)

	sportsFeeds := s.db.FeedsFromCategory(sportsCtg.APIID)
	s.Require().Len(sportsFeeds, 1)
	s.Equal("Basketball", sportsFeeds[0].Title)

	unctgCtg, ok := s.db.CategoryWithName(models.Uncategorized)
	s.Require().True(ok)

	unctgFeeds := s.db.FeedsFromCategory(unctgCtg.APIID)
	s.Require().Len(unctgFeeds, 1)
	s.Equal("Baseball", unctgFeeds[0].Title)
}

func (s *ServerTestSuite) TestOPMLExport() {
	ctg := s.db.NewCategory("Sports")

	bsktblFeed, err := s.db.NewFeedWithCategory("Basketball", "http://example.com/basketball", ctg.APIID)
	s.Require().Nil(err)

	bsblFeed := s.db.NewFeed("Baseball", "http://example.com/baseball")

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/export", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/xml")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	data, err := ioutil.ReadAll(resp.Body)
	s.Require().Nil(err)

	b := models.OPML{}
	err = xml.Unmarshal(data, &b)
	s.Require().Nil(err)

	s.Require().Len(b.Body.Items, 2)

	passed := true
	for _, item := range b.Body.Items {
		if len(item.Items) == 1 {
			if item.Title != "Sports" || len(item.Items) != 1 && item.Items[0].Title != bsktblFeed.Title {
				passed = false
				break
			}
		} else if item.Title != bsblFeed.Title {
			passed = false
			break
		}
	}

	s.True(passed)
}

func TestServerTestSuite(t *testing.T) {
	dir := http.Dir(os.Getenv("GOPATH") + "/src/github.com/varddum/syndication/server/")
	mockRSSServer = httptest.NewServer(http.FileServer(dir))
	defer mockRSSServer.Close()

	suite.Run(t, new(ServerTestSuite))
}
