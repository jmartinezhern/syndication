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
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

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

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (s *SyncTestSuite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("If-None-Match") != RSSFeedEtag {
		http.FileServer(http.Dir(os.Getenv("GOPATH")+"/src/github.com/varddum/syndication/sync/")).ServeHTTP(w, r)
	}
}

func (s *SyncTestSuite) SetupTest() {
	randUserName := RandStringRunes(8)
	s.user = s.db.NewUser(randUserName, "golang")
	s.Require().NotEmpty(s.user.APIID)
}

func (s *SyncTestSuite) TearDownTest() {
	s.db.DeleteUser(s.user.APIID)
}

func (s *SyncTestSuite) TestFeedWithNonMatchingEtag() {
	feed := s.db.NewFeed("Sync Test", "http://localhost:9090/rss.xml", &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestFeedWithMatchingEtag() {
	feed := s.db.NewFeed("Sync Test", "http://localhost:9090/rss.xml", &s.user)
	s.Require().NotEmpty(feed.APIID)

	feed.Etag = RSSFeedEtag

	err := s.db.EditFeed(&feed, &s.user)
	s.Require().Nil(err)

	err = s.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Len(entries, 0)
}

func (s *SyncTestSuite) TestFeedWithRecentLastUpdateDate() {
	feed := s.db.NewFeed("Sync Test", "http://localhost:9090/rss.xml", &s.user)
	s.Require().NotEmpty(feed.APIID)

	feed.LastUpdated = time.Now()

	err := s.db.EditFeed(&feed, &s.user)
	s.Require().Nil(err)

	err = s.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Len(entries, 0)
}

func (s *SyncTestSuite) TestFeedWithNewEntriesWithGUIDs() {
	feed := s.db.NewFeed("Sync Test", "http://localhost:9090/rss.xml", &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Require().Nil(err)
	s.Len(entries, 5)

	feed.LastUpdated = time.Time{}

	err = s.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Require().Nil(err)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestFeedWithNewEntriesWithoutGUIDs() {
	feed := s.db.NewFeed("Sync Test", "http://localhost:9090/rss_minimal.xml", &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Require().Nil(err)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncUser() {
	feed := s.db.NewFeed("Sync Test", "http://localhost:9090/rss_minimal.xml", &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.sync.SyncUser(&s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Require().Nil(err)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncUsers() {
	for i := 0; i < 10; i++ {
		user := s.db.NewUser("test"+strconv.Itoa(i), "test"+strconv.Itoa(i))

		_, found := s.db.UserWithName("test" + strconv.Itoa(i))
		s.Require().True(found)

		s.db.NewFeed("Sync Test", "http://localhost:9090/rss_minimal.xml", &user)
	}

	s.sync = NewSync(s.db, config.Sync{SyncInterval: config.Duration{Duration: time.Second * 2}})

	s.sync.SyncUsers()

	s.sync.userWaitGroup.Wait()

	users := s.db.Users()
	users = users[1:]

	for _, user := range users {
		entries := s.db.Entries(true, models.Any, &user)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) startServer() {
	s.db, _ = database.NewDB(config.Database{
		Type:       "sqlite3",
		Connection: TestDatabasePath,
	})

	s.server = &http.Server{
		Addr:    ":9090",
		Handler: s,
	}

	go s.server.ListenAndServe()

	time.Sleep(time.Second)

	s.sync = NewSync(s.db, config.Sync{SyncInterval: config.Duration{Duration: time.Second * 5}})
}

func TestSyncTestSuite(t *testing.T) {
	syncSuite := SyncTestSuite{}
	syncSuite.startServer()

	suite.Run(t, &syncSuite)

	syncSuite.db.Close()
	syncSuite.server.Close()
	os.Remove(TestDatabasePath)
}
