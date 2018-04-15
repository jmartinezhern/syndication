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
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

const (
	testDatabasePath = "/tmp/syndication-test-sync.db"

	rssFeedTag = "123456"

	feedPort = ":9090"

	baseURL = "http://localhost"

	rssMinimalURL = baseURL + feedPort + "/rss_minimal.xml"

	testSyncInterval = time.Second * 5
)

type (
	SyncTestSuite struct {
		suite.Suite

		user   models.User
		db     database.UserDB
		gDB    *database.DB
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
	if r.Header.Get("If-None-Match") != rssFeedTag {
		http.FileServer(http.Dir(os.Getenv("GOPATH")+"/src/github.com/varddum/syndication/sync/")).ServeHTTP(w, r)
	}
}

func (s *SyncTestSuite) SetupTest() {
	randUserName := RandStringRunes(8)
	s.user = s.gDB.NewUser(randUserName, "golang")
	s.db = s.gDB.NewUserDB(s.user)
	s.Require().NotEmpty(s.user.APIID)
}

func (s *SyncTestSuite) TearDownTest() {
	s.gDB.DeleteAll()
}

func (s *SyncTestSuite) TestSyncUser() {
	feed := s.db.NewFeed("Sync Test", rssMinimalURL)
	s.Require().NotEmpty(feed.APIID)

	serv := NewService(s.gDB, testSyncInterval)
	err := serv.SyncUser(&s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.MarkerAny)
	s.Require().Nil(err)
	s.Len(entries, 5)
}

func (s *SyncTestSuite) TestSyncUsers() {
	for i := 0; i < 10; i++ {
		user := s.gDB.NewUser("test"+strconv.Itoa(i), "test"+strconv.Itoa(i))

		_, found := s.gDB.UserWithName("test" + strconv.Itoa(i))
		s.Require().True(found)

		userDB := s.gDB.NewUserDB(user)

		userDB.NewFeed("Sync Test", rssMinimalURL)
	}

	serv := NewService(s.gDB, testSyncInterval)

	serv.SyncUsers()

	serv.userWaitGroup.Wait()

	users := s.gDB.Users()
	users = users[1:]

	for _, user := range users {
		userDB := s.gDB.NewUserDB(user)
		entries := userDB.Entries(true, models.MarkerAny)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) TestSyncService() {
	for i := 0; i < 10; i++ {
		user := s.gDB.NewUser("test"+strconv.Itoa(i), "test"+strconv.Itoa(i))

		_, found := s.gDB.UserWithName("test" + strconv.Itoa(i))
		s.Require().True(found)

		userDB := s.gDB.NewUserDB(user)

		userDB.NewFeed("Sync Test", rssMinimalURL)
	}

	serv := NewService(s.gDB, time.Second)

	serv.Start()

	time.Sleep(time.Second * 2)

	serv.Stop()

	users := s.gDB.Users()
	users = users[1:]

	for _, user := range users {
		userDB := s.gDB.NewUserDB(user)
		entries := userDB.Entries(true, models.MarkerAny)
		s.Len(entries, 5)
	}
}

func (s *SyncTestSuite) startServer() {
	s.gDB, _ = database.NewDB("sqlite3", testDatabasePath)

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

	syncSuite.gDB.Close()
	syncSuite.server.Close()
	os.Remove(testDatabasePath)
}
