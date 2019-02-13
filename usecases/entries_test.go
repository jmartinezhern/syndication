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

package usecases

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type EntriesSuite struct {
	suite.Suite

	entry    Entry
	unctgCtg models.Category
	user     models.User
	feed     models.Feed
}

func (t *EntriesSuite) TestEntry() {
	entry, err := database.NewEntry(models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, t.feed.APIID, t.user)
	t.Require().NoError(err)

	uEntry, err := t.entry.Entry(entry.APIID, t.user)
	t.NoError(err)
	t.Equal(entry.Title, uEntry.Title)
}

func (t *EntriesSuite) TestMissingEntry() {
	_, err := t.entry.Entry("bogus", t.user)
	t.EqualError(err, ErrEntryNotFound.Error())
}

func (t *EntriesSuite) TestEntries() {
	entry, err := database.NewEntry(models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, t.feed.APIID, t.user)
	t.Require().NoError(err)

	entries, _ := t.entry.Entries(true, models.MarkerAny, "", 2, t.user)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) TestMarkEntry() {
	entry, err := database.NewEntry(models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, t.feed.APIID, t.user)
	t.Require().NoError(err)

	entries, _ := database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Require().Empty(entries)

	err = t.entry.Mark(entry.APIID, models.MarkerRead, t.user)
	t.NoError(err)

	entries, _ = database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) TestMarkMissingEntry() {
	err := t.entry.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrEntryNotFound.Error())
}

func (t *EntriesSuite) TestMarkAll() {
	entry, err := database.NewEntry(models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, t.feed.APIID, t.user)
	t.Require().NoError(err)

	entries, _ := database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Require().Empty(entries)

	t.entry.MarkAll(models.MarkerRead, t.user)

	entries, _ = database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *EntriesSuite) SetupTest() {
	t.entry = new(EntryUsecase)

	err := database.Init("sqlite3", ":memory:")
	t.Require().NoError(err)

	t.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}
	database.CreateUser(&t.user)

	t.unctgCtg = models.Category{
		APIID: utils.CreateAPIID(),
		Name:  models.Uncategorized,
	}
	database.CreateCategory(&t.unctgCtg, t.user)

	t.feed = models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "example.com",
	}
	err = database.CreateFeed(&t.feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)
}

func (t *EntriesSuite) TearDownTest() {
	err := database.Close()
	t.NoError(err)
}

func TestEntries(t *testing.T) {
	suite.Run(t, new(EntriesSuite))
}
