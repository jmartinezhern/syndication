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

package services

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type EntriesSuite struct {
	suite.Suite

	service     Entries
	db          *sql.DB
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

	uEntry, err := t.service.Entry(entry.ID, t.user.ID)
	t.NoError(err)
	t.Equal(entry.Title, uEntry.Title)
}

func (t *EntriesSuite) TestMissingEntry() {
	_, err := t.service.Entry("bogus", t.user.ID)
	t.EqualError(err, ErrEntryNotFound.Error())
}

func (t *EntriesSuite) TestEntries() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	entries, _ := t.service.Entries("", 1, true, models.MarkerAny, t.user.ID)
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

	err := t.service.Mark(entry.ID, models.MarkerRead, t.user.ID)
	t.NoError(err)

	entries, _ := t.entriesRepo.List(t.user.ID, "", 2, true, models.MarkerRead)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) TestMarkMissingEntry() {
	err := t.service.Mark("bogus", models.MarkerRead, t.user.ID)
	t.EqualError(err, ErrEntryNotFound.Error())
}

func (t *EntriesSuite) TestMarkAll() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	t.service.MarkAll(models.MarkerRead, t.user.ID)

	entries, _ := t.entriesRepo.List(t.user.ID, "", 2, true, models.MarkerRead)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")
	t.entriesRepo = sql.NewEntries(t.db)
	t.service = NewEntriesService(t.entriesRepo)

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
