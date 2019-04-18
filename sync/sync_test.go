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
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

const (
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

		user        *models.User
		server      *http.Server
		db          *sql.DB
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

func (s *SyncTestSuite) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("If-None-Match") != rssFeedTag {
		http.FileServer(http.Dir(".")).ServeHTTP(w, r)
	}
}

func (s *SyncTestSuite) SetupTest() {
	s.db = sql.NewDB("sqlite3", ":memory:")

	s.usersRepo = sql.NewUsers(s.db)

	randUserName := RandStringRunes(8)
	s.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: randUserName,
	}
	s.usersRepo.Create(s.user)

	s.ctgsRepo = sql.NewCategories(s.db)
	s.feedsRepo = sql.NewFeeds(s.db)
	s.entriesRepo = sql.NewEntries(s.db)
}

func (s *SyncTestSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func (s *SyncTestSuite) TestPullUnreachableFeed() {
	_, _, err := utils.PullFeed("Sync Test", baseURL+feedPort+"/bogus.xml")
	s.Error(err)
}

func (s *SyncTestSuite) TestPullFeedWithBadSubscription() {
	_, _, err := utils.PullFeed("Sync Test", "bogus")
	s.Error(err)
}

func (s *SyncTestSuite) TestSyncWithEtags() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Sync Test",
		Subscription: rssURL,
	}

	s.feedsRepo.Create(s.user, &feed)

	_, entries, err := utils.PullFeed(feed.Subscription, "")
	s.Require().NoError(err)
	s.Require().Len(entries, 5)

	for idx := range entries {
		s.entriesRepo.Create(s.user, &entries[idx])
	}

	serv := NewService(testSyncInterval, 1, s.feedsRepo, s.usersRepo, s.entriesRepo)
	serv.SyncUser(s.user)

	entries, _ = s.entriesRepo.ListFromFeed(s.user, feed.APIID, "", 5, true, models.MarkerAny)
	s.Require().NoError(err)
	s.Len(entries, 0)
}

func (s *SyncTestSuite) TestSyncUser() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Sync Test",
		Subscription: rssURL,
	}
	s.feedsRepo.Create(s.user, &feed)

	serv := NewService(testSyncInterval, 1, s.feedsRepo, s.usersRepo, s.entriesRepo)
	serv.SyncUser(s.user)

	entries, _ := s.entriesRepo.ListFromFeed(s.user, feed.APIID, "", 5, true, models.MarkerAny)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncUsers() {
	for i := 0; i < 10; i++ {
		user := models.User{
			APIID:    utils.CreateAPIID(),
			Username: "test" + strconv.Itoa(i),
		}
		s.usersRepo.Create(&user)

		feed := models.Feed{
			APIID:        utils.CreateAPIID(),
			Title:        "Sync Test",
			Subscription: rssMinimalURL,
		}
		s.feedsRepo.Create(&user, &feed)
	}

	serv := NewService(testSyncInterval, 1, s.feedsRepo, s.usersRepo, s.entriesRepo)

	serv.SyncUsers()

	users, _ := s.usersRepo.List("", 11)
	users = users[1:]

	for idx := range users {
		entries, _ := s.entriesRepo.List(&users[idx], "", 100, true, models.MarkerAny)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) TestSyncService() {
	for i := 0; i < 10; i++ {
		user := models.User{
			APIID:    utils.CreateAPIID(),
			Username: "test" + strconv.Itoa(i),
		}
		s.usersRepo.Create(&user)

		feed := models.Feed{
			APIID:        utils.CreateAPIID(),
			Title:        "Sync Test",
			Subscription: rssMinimalURL,
		}
		s.feedsRepo.Create(&user, &feed)
	}

	serv := NewService(time.Second, 1, s.feedsRepo, s.usersRepo, s.entriesRepo)

	serv.Start()

	time.Sleep(time.Second * 2)

	serv.Stop()

	users, _ := s.usersRepo.List("", 10)
	users = users[1:]

	for idx := range users {
		entries, _ := s.entriesRepo.List(&users[idx], "", 100, true, models.MarkerAny)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) startServer() {
	s.server = &http.Server{
		Addr:    feedPort,
		Handler: s,
	}

	go func() {
		err := s.server.ListenAndServe()
		fmt.Println(err)
	}()

	time.Sleep(time.Second)
}

func TestSyncTestSuite(t *testing.T) {
	syncSuite := SyncTestSuite{}
	syncSuite.startServer()

	suite.Run(t, &syncSuite)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := syncSuite.server.Shutdown(ctx)
	if err != nil {
		t.Log(err)
	}
}
