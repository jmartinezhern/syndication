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

func (t *UsecasesTestSuite) TestNewFeed() {
	feed := t.feed.New("Example", "example.com", t.user)
	_, found := database.FeedWithAPIID(feed.APIID, t.user)
	t.True(found)
}

func (t *UsecasesTestSuite) TestFeeds() {
	feed := database.NewFeed("Example", "example.com", t.user)

	feeds := t.feed.Feeds(t.user)
	t.Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *UsecasesTestSuite) TestFeed() {
	feed := database.NewFeed("Example", "example.com", t.user)
	_, found := t.feed.Feed(feed.APIID, t.user)
	t.True(found)
}

func (t *UsecasesTestSuite) TestEditFeed() {
	feed := database.NewFeed("Example", "'example.com", t.user)
	newFeed, err := t.feed.Edit(feed.APIID, models.Feed{Title: "New Title"}, t.user)
	t.NoError(err)
	t.Equal("New Title", newFeed.Title)
}

func (t *UsecasesTestSuite) TestEditMissingFeed() {
	_, err := t.feed.Edit("bogus", models.Feed{}, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *UsecasesTestSuite) TestDeleteFeed() {
	feed := database.NewFeed("Example", "example.com", t.user)
	err := t.feed.Delete(feed.APIID, t.user)
	t.NoError(err)

	_, found := database.FeedWithAPIID(feed.APIID, t.user)
	t.False(found)
}

func (t *UsecasesTestSuite) TestDeleteMissingFeed() {
	err := t.feed.Delete("bogus", t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *UsecasesTestSuite) TestMarkFeed() {
	feed := database.NewFeed("Example", "example.com", t.user)

	_, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	t.Require().Empty(database.FeedEntries(feed.APIID, true, models.MarkerRead, t.user))

	t.feed.Mark(feed.APIID, models.MarkerRead, t.user)

	t.NotEmpty(database.FeedEntries(feed.APIID, true, models.MarkerRead, t.user))
}

func (t *UsecasesTestSuite) TestMarkMissingFeed() {
	err := t.feed.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *UsecasesTestSuite) TestFeedEntries() {
	feed := database.NewFeed("Example", "example.com", t.user)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	entries, err := t.feed.Entries(feed.APIID, true, models.MarkerAny, t.user)
	t.NoError(err)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *UsecasesTestSuite) TestMissingFeedEntries() {
	_, err := t.feed.Entries("bogus", true, models.MarkerAny, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *UsecasesTestSuite) TestFeedStats() {
	feed := database.NewFeed("Example", "example.com", t.user)
	_, err := t.feed.Stats(feed.APIID, t.user)
	t.NoError(err)
}

func (t *UsecasesTestSuite) TestMissingFeedStats() {
	_, err := t.feed.Stats("bogus", t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}
