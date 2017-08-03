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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/chavamee/syndication/config"
	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/models"
	"github.com/chavamee/syndication/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const TestDBPath = "/tmp/syndication-test-server.db"

type (
	ServerTestSuite struct {
		suite.Suite

		db     *database.DB
		sync   *sync.Sync
		server *Server
		user   models.User
		token  string
		ts     *httptest.Server
	}
)

func (suite *ServerTestSuite) SetupTest() {
	conf := config.DefaultConfig
	conf.Server.HTTPPort = 8080
	conf.Server.AuthSecret = "secret"

	var err error
	suite.db, err = database.NewDB("sqlite3", TestDBPath)
	suite.Require().Nil(err)

	suite.sync = sync.NewSync(suite.db)

	if suite.server == nil {
		suite.server = NewServer(suite.db, suite.sync, conf.Server)
		suite.server.handle.HideBanner = true
		go suite.server.Start()
	}

	time.Sleep(10000)

	resp, err := http.PostForm("http://localhost:8080/v1/register",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(204, resp.StatusCode)

	err = resp.Body.Close()
	suite.Nil(err)

	resp, err = http.PostForm("http://localhost:8080/v1/login",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(resp.StatusCode, 200)

	type Token struct {
		Token string `json:"token"`
	}

	var t Token
	err = json.NewDecoder(resp.Body).Decode(&t)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(t.Token)

	suite.token = t.Token

	suite.user, err = suite.db.UserWithName("GoTest")
	suite.Require().Nil(err)

	err = resp.Body.Close()
	suite.Nil(err)

	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	suite.ts = httptest.NewServer(http.HandlerFunc(handler))
}

func (suite *ServerTestSuite) TearDownTest() {
	suite.db.DeleteAll()
	suite.ts.Close()
}

func (suite *ServerTestSuite) TestNewFeed() {
	payload := []byte(`{"title":"EFF", "subscription": "https://www.eff.org/rss/updates.xml"}`)
	req, err := http.NewRequest("POST", "http://localhost:8080/v1/feeds", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respFeed.UUID)
	suite.NotEmpty(respFeed.Title)

	dbFeed, err := suite.db.Feed(respFeed.UUID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(dbFeed.Title, respFeed.Title)

}

func (suite *ServerTestSuite) TestGetFeeds() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			Title:        "Feed " + strconv.Itoa(i+1),
			Subscription: "http://example.com/feed",
		}
		err := suite.db.NewFeed(&feed, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(feed.ID)
		suite.Require().NotEmpty(feed.UUID)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	suite.Require().Nil(err)
	suite.Len(respFeeds.Feeds, 5)
}

func (suite *ServerTestSuite) TestGetFeed() {
	feed := models.Feed{
		Title:        "EFF",
		Subscription: "https://www.eff.org/rss/updates.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds/"+feed.UUID, nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	suite.Require().Nil(err)

	suite.Equal(respFeed.Title, feed.Title)
	suite.Equal(respFeed.UUID, feed.UUID)
}

func (suite *ServerTestSuite) TestEditFeed() {
	feed := models.Feed{Title: "EFF", Subscription: "https://www.eff.org/rss/updates.xml"}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	payload := []byte(`{"title": "EFF Updates"}`)
	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/feeds/"+feed.UUID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	respFeed, err := suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(respFeed.Title, "EFF Updates")
}

func (suite *ServerTestSuite) TestDeleteFeed() {
	feed := models.Feed{Title: "EFF", Subscription: "https://www.eff.org/rss/updates.xml"}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	req, err := http.NewRequest("DELETE", "http://localhost:8080/v1/feeds/"+feed.UUID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Feed(feed.UUID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestGetEntriesFromFeed() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds/"+feed.UUID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestMarkFeed() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/feeds/"+feed.UUID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)
}

func (suite *ServerTestSuite) TestNewCategory() {
	payload := []byte(`{"name": "News"}`)
	req, err := http.NewRequest("POST", "http://localhost:8080/v1/categories", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respCtg.UUID)
	suite.NotEmpty(respCtg.Name)

	dbCtg, err := suite.db.Category(respCtg.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(dbCtg.Name, respCtg.Name)
}

func (suite *ServerTestSuite) TestGetCategories() {
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			Name: "Category " + strconv.Itoa(i+1),
		}
		err := suite.db.NewCategory(&ctg, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(ctg.ID)
		suite.Require().NotEmpty(ctg.UUID)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Categories struct {
		Categories []models.Category `json:"categories"`
	}

	respCtgs := new(Categories)
	err = json.NewDecoder(resp.Body).Decode(respCtgs)
	suite.Require().Nil(err)

	suite.Len(respCtgs.Categories, 7)
}

func (suite *ServerTestSuite) TestGetCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.UUID)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+ctg.UUID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	suite.Require().Nil(err)

	suite.Equal(respCtg.Name, ctg.Name)
	suite.Equal(respCtg.UUID, ctg.UUID)
}

func (suite *ServerTestSuite) TestEditCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.UUID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/categories/"+ctg.UUID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	editedCtg, err := suite.db.Category(ctg.UUID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(editedCtg.Name, "World News")
}

func (suite *ServerTestSuite) TestDeleteCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.UUID)

	req, err := http.NewRequest("DELETE", "http://localhost:8080/v1/categories/"+ctg.UUID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Category(ctg.UUID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestGetFeedsFromCategory() {
	category := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)

	feed := models.Feed{
		Title:        "Test feed",
		Subscription: "http://localhost:8080",
		Category:     category,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+category.UUID+"/feeds", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	suite.Require().Nil(err)
	suite.Len(respFeeds.Feeds, 1)
}

func (suite *ServerTestSuite) TestGetEntriesFromCategory() {
	category := models.Category{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)
	suite.Require().NotZero(category.ID)

	feed := models.Feed{
		Subscription: suite.ts.URL,
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncCategory(&category, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+category.UUID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestMarkCategory() {
	category := models.Category{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)

	feed := models.Feed{
		Subscription: suite.ts.URL,
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncCategory(&category, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromCategory(category.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	entries, err = suite.db.EntriesFromCategory(category.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/categories/"+category.UUID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromCategory(category.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromCategory(category.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)
}

func (suite *ServerTestSuite) TestGetEntries() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestGetEntry() {
	feed := models.Feed{
		Title:        "EFF",
		Subscription: "https://www.eff.org/rss/updates.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	entry := models.Entry{
		Title:  "The Espionage Acts Troubling Origins",
		Link:   "https://www.eff.org/deeplinks/2017/06/one-hundred-years-espionage-act",
		Feed:   feed,
		FeedID: feed.ID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(entry.ID)
	suite.Require().NotEmpty(entry.UUID)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/entries/"+entry.UUID, nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respEntry := new(models.Entry)
	err = json.NewDecoder(resp.Body).Decode(respEntry)
	suite.Require().Nil(err)

	suite.Equal(entry.Title, respEntry.Title)
	suite.Equal(entry.UUID, respEntry.UUID)
}

func (suite *ServerTestSuite) TestMarkEntry() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/entries/"+entries[0].UUID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 4)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)
}

func (suite *ServerTestSuite) TestGetStatsForFeed() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Read,
			Saved:  true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds/"+feed.UUID+"/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestGetStats() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Read,
			Saved:  true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/entries/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestGetStatsForCategory() {
	category := models.Category{
		Name: "World",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)

	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Read,
			Saved:  true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+category.UUID+"/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestAddFeedsToCategory() {

}

func TestServerRegister(t *testing.T) {
	conf := config.DefaultConfig
	conf.Server.HTTPPort = 8060

	db, err := database.NewDB("sqlite3", "/tmp/syndication-test-server-register.db")
	defer os.Remove(db.Connection)
	require.NotNil(t, db)
	require.Nil(t, err)

	sync := sync.NewSync(db)
	require.NotNil(t, sync)

	server := NewServer(db, sync, conf.Server)
	server.handle.HideBanner = true

	go func() {
		server.Start()
	}()

	time.Sleep(1000)

	regResp, err := http.PostForm("http://localhost:8060/v1/register",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	require.Nil(t, err)

	assert.Equal(t, 204, regResp.StatusCode)

	users := db.Users("username")
	assert.Len(t, users, 1)

	assert.Equal(t, "GoTest", users[0].Username)
	assert.NotEmpty(t, users[0].ID)
	assert.NotEmpty(t, users[0].UUID)

	err = regResp.Body.Close()
	require.Nil(t, err)

	server.Stop()
}

func TestServerLogin(t *testing.T) {
	conf := config.DefaultConfig
	conf.Server.HTTPPort = 8070

	db, err := database.NewDB("sqlite3", "/tmp/syndication-test-server-login.db")
	require.Nil(t, err)
	defer os.Remove(db.Connection)

	sync := sync.NewSync(db)

	server := NewServer(db, sync, conf.Server)
	server.handle.HideBanner = true

	go func() {
		server.Start()
	}()

	time.Sleep(1000)

	regResp, err := http.PostForm("http://localhost:8070/v1/register",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	require.Nil(t, err)

	assert.Equal(t, 204, regResp.StatusCode)

	err = regResp.Body.Close()
	require.Nil(t, err)

	loginResp, err := http.PostForm("http://localhost:8070/v1/login",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	require.Nil(t, err)

	assert.Equal(t, 200, loginResp.StatusCode)

	type Token struct {
		Token string `json:"token"`
	}

	var token Token
	err = json.NewDecoder(loginResp.Body).Decode(&token)
	require.Nil(t, err)
	assert.NotEmpty(t, token.Token)

	_, err = db.UserWithName("GoTest")
	assert.Nil(t, err)

	err = loginResp.Body.Close()
	assert.Nil(t, err)

	server.Stop()
}

func TestServerTestSuite(t *testing.T) {
	serverSuite := new(ServerTestSuite)
	suite.Run(t, serverSuite)
	serverSuite.server.Stop()
	os.Remove(TestDBPath)
}
