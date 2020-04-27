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
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

const (
	rssFile = `
	<rss>
	  <channel>
	    <title>RSS Test</title>
	    <link>http://localhost:8090</link>
	    <author>webmaster@example.com</author>
	    <description>Testing rss feeds</description>
	    <language>en</language>
	    <lastBuildDate></lastBuildDate>
	    <item>
	      <title>Item 1</title>
	      <link>http://localhost:8090/item_1</link>
	      <description>Single test item</description>
	      <author>jmartinezhern</author>
	      <guid>item1@test</guid>
	      <pubDate>Sun, 19 May 2002 15:21:36 GMT</pubDate>
	      <source>http://localhost:8090/rss.xml</source>
	    </item>
	    <item>
	      <title>Item 2</title>
	      <link>http://localhost:8090/item_2</link>
	      <description>Single test item</description>
	      <author>jmartinezhern</author>
	      <guid>item2@test</guid>
	      <pubDate>Sun, 19 May 2002 15:21:36 GMT</pubDate>
	      <source>http://localhost:8090/rss.xml</source>
	    </item>
	    <item>
	      <title>Item 3</title>
	      <link>http://localhost:8090/item_3</link>
	      <description>Single test item</description>
	      <author>jmartinezhern</author>
	      <guid>item3@test</guid>
	      <pubDate>Sun, 19 May 2002 15:21:36 GMT</pubDate>
	      <source>http://localhost:8090/rss.xml</source>
	    </item>
	    <item>
	      <title>Item 4</title>
	      <link>http://localhost:8090/item_4</link>
	      <description>Single test item</description>
	      <author>jmartinezhern</author>
	      <guid>item4@test</guid>
	      <pubDate>Sun, 19 May 2002 15:21:36 GMT</pubDate>
	      <source>http://localhost:8090/rss.xml</source>
	    </item>
	    <item>
	      <title>Item 5</title>
	      <link>http://localhost:8090/item_5</link>
	      <description>Single test item</description>
	      <author>jmartinezhern</author>
	      <guid>item5@test</guid>
	      <pubDate>Sun, 19 May 2002 15:21:36 GMT</pubDate>
	      <source>http://localhost:8090/rss.xml</source>
	    </item>
	  </channel>
	</rss>
	`

	rssMinimalFile = `
	<rss>
	  <channel>
	    <title>Science News of yesterday, today!</title>
	    <link>https://example.com/news</link>
	    <description>Yesterday's news in science delivered today</description>
	    <item>
	      <title>Item 1</title>
	    </item>
	    <item>
	      <title>Item 2</title>
	    </item>
	    <item>
	      <title>Item 3</title>
	    </item>
	    <item>
	      <title>Item 4</title>
	    </item>
	    <item>
	      <title>Item 5</title>
	    </item>
	  </channel>
	</rss>
	`

	rssFeedTag       = "123456"
	testSyncInterval = time.Second * 5
)

type (
	SyncTestSuite struct {
		suite.Suite

		ts          *httptest.Server
		db          *gorm.DB
		ctgsRepo    repo.Categories
		feedsRepo   repo.Feeds
		usersRepo   repo.Users
		entriesRepo repo.Entries
	}
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)

	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}

func (s *SyncTestSuite) SetupTest() {
	var err error

	s.db, err = gorm.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	sql.AutoMigrateTables(s.db)

	s.usersRepo = sql.NewUsers(s.db)

	s.ctgsRepo = sql.NewCategories(s.db)
	s.feedsRepo = sql.NewFeeds(s.db)
	s.entriesRepo = sql.NewEntries(s.db)
}

func (s *SyncTestSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func (s *SyncTestSuite) TestPullUnreachableFeed() {
	_, _, err := utils.PullFeed("Sync Test", s.ts.URL+"/bogus.xml")
	s.Error(err)
}

func (s *SyncTestSuite) TestPullFeedWithBadSubscription() {
	_, _, err := utils.PullFeed("Sync Test", "bogus")
	s.Error(err)
}

func (s *SyncTestSuite) TestSyncWithEtags() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Sync Test",
		Subscription: s.ts.URL + "/rss.xml",
	}

	user := &models.User{
		ID:       utils.CreateID(),
		Username: RandStringRunes(8),
	}
	s.usersRepo.Create(user)

	s.feedsRepo.Create(user.ID, &feed)

	_, entries, err := utils.PullFeed(feed.Subscription, "")
	s.Require().NoError(err)
	s.Require().Len(entries, 5)

	for idx := range entries {
		s.entriesRepo.Create(user.ID, &entries[idx])
	}

	serv := NewService(testSyncInterval, s.feedsRepo, s.usersRepo, s.entriesRepo)
	serv.syncUser(user.ID)

	entries, _ = s.entriesRepo.ListFromFeed(user.ID, models.Page{
		FilterID:       feed.ID,
		ContinuationID: "",
		Count:          5,
		Newest:         true,
		Marker:         models.MarkerAny,
	})
	s.Len(entries, 0)
}

func (s *SyncTestSuite) TestSyncUser() {
	user := &models.User{
		ID:       utils.CreateID(),
		Username: RandStringRunes(8),
	}
	s.usersRepo.Create(user)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Sync Test",
		Subscription: s.ts.URL + "/rss.xml",
	}
	s.feedsRepo.Create(user.ID, &feed)

	serv := NewService(testSyncInterval, s.feedsRepo, s.usersRepo, s.entriesRepo)
	serv.syncUser(user.ID)

	entries, _ := s.entriesRepo.ListFromFeed(user.ID, models.Page{
		FilterID:       feed.ID,
		ContinuationID: "",
		Count:          5,
		Newest:         true,
		Marker:         models.MarkerAny,
	})
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncService() {
	serv := NewService(time.Second, s.feedsRepo, s.usersRepo, s.entriesRepo)

	serv.Start()

	serv.Stop()
}

func TestSyncTestSuite(t *testing.T) {
	syncSuite := SyncTestSuite{}

	syncSuite.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp string

		match := r.Header.Get("If-None-Match")

		if match == rssFeedTag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		switch r.URL.Path {
		case "/rss.xml":
			resp = rssFile
		case "/rss_minimal.xml":
			resp = rssMinimalFile
		default:
			resp = ""
		}

		if _, err := fmt.Fprint(w, resp); err != nil {
			panic(err)
		}
	}))
	defer syncSuite.ts.Close()

	suite.Run(t, &syncSuite)
}
