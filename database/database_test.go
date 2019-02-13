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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	DatabaseTestSuite struct {
		suite.Suite

		user models.User
	}
)

func (s *DatabaseTestSuite) SetupTest() {
	err := Init("sqlite3", ":memory:")

	s.Require().NoError(err)

	s.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test",
	}
	CreateUser(&s.user)
}

func (s *DatabaseTestSuite) TearDownTest() {
	err := Close()
	s.NoError(err)
}

func TestNewDB(t *testing.T) {
	_, err := NewDB("sqlite3", ":memory:")
	assert.NoError(t, err)
}

func TestNewDBWithBadOptions(t *testing.T) {
	_, err := NewDB("bogus", ":memory:")
	assert.Error(t, err)
}

func (s *DatabaseTestSuite) TestStats() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  models.Uncategorized,
	}, s.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := CreateFeed(&feed, ctgID, s.user)
	s.Require().NoError(err)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerRead,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err := NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)
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
