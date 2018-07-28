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
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

func (t *UsecasesTestSuite) TestEntry() {
	feed := database.NewFeed("Example", "example.com", t.user)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	uEntry, err := t.entry.Entry(entry.APIID, t.user)
	t.NoError(err)
	t.Equal(entry.Title, uEntry.Title)
}

func (t *UsecasesTestSuite) TestMissingEntry() {
	_, err := t.entry.Entry("bogus", t.user)
	t.EqualError(err, ErrEntryNotFound.Error())
}

func (t *UsecasesTestSuite) TestEntries() {
	feed := database.NewFeed("Example", "example.com", t.user)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	entries := t.entry.Entries(true, models.MarkerAny, t.user)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *UsecasesTestSuite) TestMarkEntry() {
	feed := database.NewFeed("Example", "example.com", t.user)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	t.Require().Empty(database.Entries(true, models.MarkerRead, t.user))

	err = t.entry.Mark(entry.APIID, models.MarkerRead, t.user)
	t.NoError(err)
	entries := database.Entries(true, models.MarkerRead, t.user)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *UsecasesTestSuite) TestMarkMissingEntry() {
	err := t.entry.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrEntryNotFound.Error())
}
