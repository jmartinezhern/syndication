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

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

const (
	testDatabasePath = "/tmp/syndication-test-sync.db"

	rssFeedTag = "123456"

	feedPort = ":9090"

	baseURL = "http://localhost"

	rssMinimalURL = baseURL + feedPort + "/rss_minimal.xml"

	rssURL = baseURL + feedPort + "/rss.xml"

	testSyncInterval = time.Second * 5
)

type (
	SyncTestSuite struct {
		suite.Suite

		user     models.User
		server   *http.Server
		unctgCtg models.Category
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
	if r.Header.Get("If-None-Match") != rssFeedTag {
		http.FileServer(http.Dir(".")).ServeHTTP(w, r)
	}
}

func (s *SyncTestSuite) SetupTest() {
	_ = database.Init("sqlite3", testDatabasePath)

	randUserName := RandStringRunes(8)
	s.user = database.NewUser(randUserName, "golang")
	s.unctgCtg = database.NewCategory(models.Uncategorized, s.user)
	s.Require().NotEmpty(s.user.APIID)
}

func (s *SyncTestSuite) TearDownTest() {
	os.Remove(testDatabasePath)
}

func (s *SyncTestSuite) TestPullUnreachableFeed() {
	_, _, err := PullFeed("Sync Test", baseURL+feedPort+"/bogus.xml")
	s.Error(err)
}

func (s *SyncTestSuite) TestPullFeedWithBadSubscription() {
	_, _, err := PullFeed("Sync Test", "bogus")
	s.Error(err)
}

func (s *SyncTestSuite) TestSyncWithEtags() {
	feed, err := database.NewFeed("Sync Test", rssURL, s.unctgCtg.APIID, s.user)
	s.Require().NoError(err)
	s.Require().NotEmpty(feed.APIID)

	_, entries, err := PullFeed(feed.Subscription, "")
	s.Require().NoError(err)
	s.Require().Len(entries, 5)

	_, err = database.NewEntries(entries, feed.APIID, s.user)
	s.Require().Nil(err)

	serv := NewService(testSyncInterval, 1)
	err = serv.SyncUser(&s.user)
	s.Require().Nil(err)

	entries = database.FeedEntries(feed.APIID, true, models.MarkerAny, s.user)
	s.Require().Nil(err)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncUser() {
	feed, err := database.NewFeed("Sync Test", rssMinimalURL, s.unctgCtg.APIID, s.user)
	s.Require().NoError(err)
	s.Require().NotEmpty(feed.APIID)

	serv := NewService(testSyncInterval, 1)
	err = serv.SyncUser(&s.user)
	s.Require().Nil(err)

	entries := database.FeedEntries(feed.APIID, true, models.MarkerAny, s.user)
	s.Require().Nil(err)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncUsers() {
	for i := 0; i < 10; i++ {
		user := database.NewUser("test"+strconv.Itoa(i), "test"+strconv.Itoa(i))

		_, found := database.UserWithName("test" + strconv.Itoa(i))
		s.Require().True(found)

		ctg := database.NewCategory(models.Uncategorized, user)
		_, err := database.NewFeed("Sync Test", rssMinimalURL, ctg.APIID, user)
		s.Require().NoError(err)
	}

	serv := NewService(testSyncInterval, 1)

	serv.SyncUsers()

	serv.userWaitGroup.Wait()

	users := database.Users()[1:]

	for _, user := range users {
		entries, _ := database.Entries(true, models.MarkerAny, "", 100, user)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) TestSyncService() {
	for i := 0; i < 10; i++ {
		user := database.NewUser("test"+strconv.Itoa(i), "test"+strconv.Itoa(i))

		_, found := database.UserWithName("test" + strconv.Itoa(i))
		s.Require().True(found)

		ctg := database.NewCategory(models.Uncategorized, user)
		_, err := database.NewFeed("Sync Test", rssMinimalURL, ctg.APIID, user)
		s.Require().NoError(err)
	}

	serv := NewService(time.Second, 1)

	serv.Start()

	time.Sleep(time.Second * 2)

	serv.Stop()

	users := database.Users()[1:]

	for _, user := range users {
		entries, _ := database.Entries(true, models.MarkerAny, "", 100, user)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) startServer() {
	s.server = &http.Server{
		Addr:    feedPort,
		Handler: s,
	}

	go s.server.ListenAndServe()

	time.Sleep(time.Second)
}

func TestSyncTestSuite(t *testing.T) {
	syncSuite := SyncTestSuite{}
	syncSuite.startServer()

	suite.Run(t, &syncSuite)

	database.Close()
	syncSuite.server.Close()
	os.Remove(testDatabasePath)
}
