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

package services_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type EntriesSuite struct {
	suite.Suite

	service     services.Entries
	db          *gorm.DB
	entriesRepo repo.Entries
	feedsRepo   repo.Feeds
	user        *models.User
	feed        models.Feed
}

func (t *EntriesSuite) TestEntry() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	uEntry, err := t.service.Entry(t.user.ID, entry.ID)
	t.NoError(err)
	t.Equal(entry.Title, uEntry.Title)
}

func (t *EntriesSuite) TestMissingEntry() {
	_, err := t.service.Entry(t.user.ID, "bogus")
	t.EqualError(err, services.ErrEntryNotFound.Error())
}

func (t *EntriesSuite) TestEntries() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	entries, _ := t.service.Entries(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
		Newest:         true,
		Marker:         models.MarkerAny,
	})
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) TestMarkEntry() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	err := t.service.Mark(t.user.ID, entry.ID, models.MarkerRead)
	t.NoError(err)

	entries, _ := t.entriesRepo.List(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
		Newest:         true,
		Marker:         models.MarkerRead,
	})
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) TestMarkMissingEntry() {
	err := t.service.Mark(t.user.ID, "bogus", models.MarkerRead)
	t.EqualError(err, services.ErrEntryNotFound.Error())
}

func (t *EntriesSuite) TestMarkAll() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	t.service.MarkAll(t.user.ID, models.MarkerRead)

	entries, _ := t.entriesRepo.List(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
		Newest:         true,
		Marker:         models.MarkerRead,
	})
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) SetupTest() {
	var err error

	t.db, err = gorm.Open("sqlite3", ":memory:")
	t.Require().NoError(err)

	sql.AutoMigrateTables(t.db)

	t.entriesRepo = sql.NewEntries(t.db)
	t.service = services.NewEntriesService(t.entriesRepo)

	t.user = &models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}
	sql.NewUsers(t.db).Create(t.user)

	t.feed = models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "example.com",
	}
	t.feedsRepo = sql.NewFeeds(t.db)
	t.feedsRepo.Create(t.user.ID, &t.feed)
}

func (t *EntriesSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestEntries(t *testing.T) {
	suite.Run(t, new(EntriesSuite))
}
