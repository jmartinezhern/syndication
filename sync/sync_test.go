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

package sync

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/stretchr/testify/suite"
	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

const TestDatabasePath = "/tmp/syndication-test-sync.db"

const RSSFeedEtag = "123456"

type (
	SyncTestSuite struct {
		suite.Suite

		user   models.User
		db     *database.DB
		sync   *Sync
		server *http.Server
	}
)

func (suite *SyncTestSuite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("If-None-Match") != RSSFeedEtag {
		http.FileServer(http.Dir(os.Getenv("GOPATH")+"/src/github.com/varddum/syndication/sync/")).ServeHTTP(w, r)
	}
}

func (suite *SyncTestSuite) SetupTest() {
	var err error
	suite.db, err = database.NewDB("sqlite3", TestDatabasePath)
	suite.Require().Nil(err)

	err = suite.db.NewUser("test", "golang")
	suite.Require().Nil(err)

	suite.user, err = suite.db.UserWithName("test")
	suite.Require().Nil(err)

	suite.server = &http.Server{
		Addr:    ":9090",
		Handler: suite,
	}

	go suite.server.ListenAndServe()

	time.Sleep(time.Second * 5)

	suite.sync = NewSync(suite.db, config.Sync{SyncInterval: time.Minute * 5})
}

func (suite *SyncTestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Connection)
	suite.server.Close()
}

func (suite *SyncTestSuite) TestFetchFeed() {
	f, err := ioutil.ReadFile(os.Getenv("GOPATH") + "/src/github.com/varddum/syndication/sync/rss.xml")
	suite.Require().Nil(err)

	fp := gofeed.NewParser()
	originalFeed, err := fp.Parse(bytes.NewReader(f))

	suite.Require().Nil(err)

	feed := &models.Feed{
		Subscription: "http://localhost:9090/rss.xml",
	}
	err = FetchFeed(feed)
	suite.Require().Nil(err)

	suite.Equal(originalFeed.Title, feed.Title)
	suite.Equal(originalFeed.Link, feed.Source)
}

func (suite *SyncTestSuite) TestFeedWithNonMatchingEtag() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestFeedWithMatchingEtag() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss.xml",
		Etag:         RSSFeedEtag,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 0)
}

func (suite *SyncTestSuite) TestFeedWithRecentLastUpdateDate() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss.xml",
		LastUpdated:  time.Now(),
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 0)
}

func (suite *SyncTestSuite) TestFeedWithNewEntriesWithGUIDs() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)

	feed.LastUpdated = time.Time{}

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestFeedWithNewEntriesWithoutGUIDs() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss_minimal.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestSyncUser() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss_minimal.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.sync.SyncUser(&suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestSyncUsers() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:9090/rss_minimal.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	suite.sync = NewSync(suite.db, config.Sync{SyncInterval: time.Second * 5})

	suite.sync.Start()

	time.Sleep(time.Second * 6)

	suite.sync.Stop()

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestUserThreadAllocation() {
	for i := 0; i < 150; i++ {
		err := suite.db.NewUser("test"+strconv.Itoa(i), "test"+strconv.Itoa(i))
		suite.Require().Nil(err)

		user, err := suite.db.UserWithName("test" + strconv.Itoa(i))
		suite.Require().Nil(err)

		feed := models.Feed{
			Title:        "Sync Test",
			Subscription: "http://localhost:9090/rss_minimal.xml",
		}

		err = suite.db.NewFeed(&feed, &user)
		suite.Require().Nil(err)
	}

	suite.sync = NewSync(suite.db, config.Sync{SyncInterval: time.Second * 5})

	suite.sync.Start()

	time.Sleep(time.Second * 6)

	suite.sync.Stop()

	users := suite.db.Users()
	users = users[1:]

	for _, user := range users {
		entries, err := suite.db.Entries(true, models.Any, &user)
		suite.Require().Nil(err)
		if len(entries) == 0 {
			fmt.Println(user)
		}
		suite.Len(entries, 5)
	}
}

func TestSyncTestSuite(t *testing.T) {
	suite.Run(t, new(SyncTestSuite))
}
