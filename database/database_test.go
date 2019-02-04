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

package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
)

type (
	DatabaseTestSuite struct {
		suite.Suite

		user models.User
	}
)

const TestDatabasePath = "/tmp/syndication-test-db.db"

func (s *DatabaseTestSuite) SetupTest() {
	err := Init("sqlite3", TestDatabasePath)

	s.Require().Nil(err)

	s.user = NewUser("test", "golang")
	s.Require().NotZero(s.user.ID)
}

func (s *DatabaseTestSuite) TearDownTest() {
	err := Close()
	s.Nil(err)

	err = os.Remove(TestDatabasePath)
	s.Nil(err)
}

func TestNewDB(t *testing.T) {
	_, err := NewDB("sqlite3", TestDatabasePath)
	assert.Nil(t, err)
	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestNewDBWithBadOptions(t *testing.T) {
	_, err := NewDB("bogus", TestDatabasePath)
	assert.NotNil(t, err)
}

func (s *DatabaseTestSuite) TestStats() {
	feed := NewFeed("News", "http://example.com", s.user)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerRead,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := NewEntry(entry, feed.APIID, s.user)
		s.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err := NewEntry(entry, feed.APIID, s.user)
		s.Require().Nil(err)
	}

	stats := Stats(s.user)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
