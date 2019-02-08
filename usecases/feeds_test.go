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
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

func (t *UsecasesTestSuite) TestNewFeed() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<rss></rss>")
	}))
	defer ts.Close()

	feed, err := t.feed.New("Example", ts.URL, t.unctgCtg.APIID, t.user)
	t.NoError(err)
	_, found := database.FeedWithAPIID(feed.APIID, t.user)
	t.True(found)
}

func (t *UsecasesTestSuite) TestUnreachableNewFeed() {
	_, err := t.feed.New("Example", "bogus", t.unctgCtg.APIID, t.user)
	t.EqualError(err, ErrFetchingFeed.Error())
}

func (t *UsecasesTestSuite) TestFeeds() {
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	feeds, _ := t.feed.Feeds("", 2, t.user)
	t.Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *UsecasesTestSuite) TestFeed() {
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)
	_, found := t.feed.Feed(feed.APIID, t.user)
	t.True(found)
}

func (t *UsecasesTestSuite) TestEditFeed() {
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	newFeed, err := t.feed.Edit(feed.APIID, models.Feed{Title: "New Title"}, t.user)
	t.NoError(err)
	t.Equal("New Title", newFeed.Title)
}

func (t *UsecasesTestSuite) TestEditMissingFeed() {
	_, err := t.feed.Edit("bogus", models.Feed{}, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *UsecasesTestSuite) TestDeleteFeed() {
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	err = t.feed.Delete(feed.APIID, t.user)
	t.NoError(err)

	_, found := database.FeedWithAPIID(feed.APIID, t.user)
	t.False(found)
}

func (t *UsecasesTestSuite) TestDeleteMissingFeed() {
	err := t.feed.Delete("bogus", t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *UsecasesTestSuite) TestMarkFeed() {
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	_, err = database.NewEntry(models.Entry{
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
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

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
	feed, err := database.NewFeed("Example", "example.com", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	_, err = t.feed.Stats(feed.APIID, t.user)
	t.NoError(err)
}

func (t *UsecasesTestSuite) TestMissingFeedStats() {
	_, err := t.feed.Stats("bogus", t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}
