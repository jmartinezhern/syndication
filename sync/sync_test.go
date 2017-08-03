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
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/models"
	"github.com/stretchr/testify/suite"
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
		http.FileServer(http.Dir(os.Getenv("GOPATH")+"/src/github.com/chavamee/syndication/sync/")).ServeHTTP(w, r)
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
		Addr:    ":8090",
		Handler: suite,
	}

	go func() {
		suite.server.ListenAndServe()
	}()

	suite.sync = NewSync(suite.db)
}

func (suite *SyncTestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Connection)
	suite.server.Close()
}

func (suite *SyncTestSuite) TestFetchFeed() {
	f, err := ioutil.ReadFile(os.Getenv("GOPATH") + "/src/github.com/chavamee/syndication/sync/rss.xml")
	suite.Require().Nil(err)

	fp := gofeed.NewParser()
	originalFeed, err := fp.Parse(bytes.NewReader(f))

	suite.Require().Nil(err)

	feed := &models.Feed{
		Subscription: "http://localhost:8090/rss.xml",
	}
	err = FetchFeed(feed)
	suite.Require().Nil(err)

	suite.Equal(originalFeed.Title, feed.Title)
	suite.Equal(originalFeed.Link, feed.Source)
}

func (suite *SyncTestSuite) TestFeedWithNonMatchingEtag() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:8090/rss.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestFeedWithMatchingEtag() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:8090/rss.xml",
		Etag:         RSSFeedEtag,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 0)
}

func (suite *SyncTestSuite) TestFeedWithLastBuildDate() {
}

func (suite *SyncTestSuite) TestFeedWithNewEntriesWithGUIDs() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:8090/rss.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)

	feed.LastUpdated = time.Time{}

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func (suite *SyncTestSuite) TestFeedWithNewEntriesWithoutGUIDs() {
	feed := models.Feed{
		Title:        "Sync Test",
		Subscription: "http://localhost:8090/rss_minimal.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Len(entries, 5)
}

func TestSyncTestSuite(t *testing.T) {
	suite.Run(t, new(SyncTestSuite))
}
