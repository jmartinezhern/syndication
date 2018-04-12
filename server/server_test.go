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
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/plugins"
)

const (
	TestDBPath = "/tmp/syndication-test-server.db"

	testHTTPPort = 9876
)

var (
	testBaseURL = "http://localhost:" + strconv.Itoa(testHTTPPort)
)

type (
	ServerTestSuite struct {
		suite.Suite

		gDB    *database.DB
		db     database.UserDB
		server *Server
		user   models.User
		token  string
		ts     *httptest.Server
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
	randUserName := RandStringRunes(8)

	user := s.gDB.NewUser(randUserName, "testtesttest")
	s.db = s.gDB.NewUserDB(user)

	s.Require().NotEmpty(user.APIID)

	token, err := s.db.NewAPIKey(s.server.config.AuthSecret, time.Hour*72)
	s.Require().Nil(err)
	s.Require().NotEmpty(token.Key)

	s.token = token.Key
	s.user = user
}

func (s *ServerTestSuite) TearDownTest() {
	s.gDB.DeleteUser(s.user.APIID)
}

func (s *ServerTestSuite) TestPlugins() {
	req, err := http.NewRequest("GET", "http://localhost:9876/api_test/hello_world", nil)
	s.Require().Nil(err)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)
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

func (s *ServerTestSuite) TestRegister() {
	s.gDB.DeleteUser(s.user.APIID)

	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(204, regResp.StatusCode)

	users := s.gDB.Users("username")
	s.Len(users, 1)

	s.Equal(randUserName, users[0].Username)
	s.NotEmpty(users[0].APIID)

	err = regResp.Body.Close()
	s.Nil(err)

	s.gDB.DeleteUser(users[0].APIID)
}

func (s *ServerTestSuite) TestLogin() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(204, regResp.StatusCode)

	err = regResp.Body.Close()
	s.Require().Nil(err)

	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(200, loginResp.StatusCode)

	type Token struct {
		Token string `json:"token"`
	}

	var token Token
	err = json.NewDecoder(loginResp.Body).Decode(&token)
	s.Require().Nil(err)
	s.NotEmpty(token.Token)

	user, found := s.gDB.UserWithName(randUserName)
	s.True(found)

	err = loginResp.Body.Close()
	s.Nil(err)

	s.gDB.DeleteUser(user.APIID)
}

func (s *ServerTestSuite) TestLoginWithNonExistentUser() {
	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {"bogus"}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(401, loginResp.StatusCode)

	err = loginResp.Body.Close()
	s.Nil(err)
}

func (s *ServerTestSuite) TestLoginWithBadPassword() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	user, found := s.gDB.UserWithName(randUserName)
	s.Require().True(found)
	defer s.gDB.DeleteUser(user.APIID)

	s.Equal(204, regResp.StatusCode)

	err = regResp.Body.Close()
	s.Require().Nil(err)

	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"bogus"}})
	s.Require().Nil(err)

	s.Equal(401, loginResp.StatusCode)

	err = loginResp.Body.Close()
	s.Nil(err)
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

func (s *ServerTestSuite) startServer() {
	conf := config.DefaultConfig
	conf.Server.HTTPPort = testHTTPPort
	conf.Server.AuthSecret = "secret"
	conf.Server.EnableRequestLogs = false

	var err error
	s.gDB, err = database.NewDB(config.Database{
		Type:       "sqlite3",
		Connection: TestDBPath,
		APIKeyExpiration: config.Duration{
			Duration: time.Hour * 72,
		},
	})
	s.Require().Nil(err)

	if s.server == nil {
		plgnPath := []string{os.Getenv("GOPATH") + "/src/github.com/varddum/syndication/api.so"}
		plgns := plugins.NewPlugins(plgnPath)

		s.server = NewServer(s.gDB, &plgns, conf.Server)
		s.server.handle.HideBanner = true
		go s.server.Start()
	}

	time.Sleep(time.Second)

	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:9876</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:9876/item_1</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:9876/item_2</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:9876/item_3</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:9876/item_4</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:9876/item_5</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	s.ts = httptest.NewServer(http.HandlerFunc(handler))
}

func TestServerTestSuite(t *testing.T) {
	serverSuite := new(ServerTestSuite)
	serverSuite.startServer()
	suite.Run(t, serverSuite)
	serverSuite.server.Stop()
	os.Remove(TestDBPath)
	serverSuite.ts.Close()
}
