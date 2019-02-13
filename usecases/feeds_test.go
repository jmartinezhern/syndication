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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type FeedsSuite struct {
	suite.Suite

	usecase  Feed
	unctgCtg models.Category
	user     models.User
	feed     models.Feed
}

func (t *FeedsSuite) TestNewFeed() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<rss></rss>")
	}))
	defer ts.Close()

	feed, err := t.usecase.New("Example", ts.URL, t.unctgCtg.APIID, t.user)
	t.NoError(err)
	_, found := database.FeedWithAPIID(feed.APIID, t.user)
	t.True(found)
}

func (t *FeedsSuite) TestUnreachableNewFeed() {
	_, err := t.usecase.New("Example", "bogus", t.unctgCtg.APIID, t.user)
	t.EqualError(err, ErrFetchingFeed.Error())
}

func (t *FeedsSuite) TestFeeds() {
	feeds, _ := t.usecase.Feeds("", 2, t.user)
	t.Len(feeds, 1)
	t.Equal(t.feed.Title, feeds[0].Title)
}

func (t *FeedsSuite) TestFeed() {
	_, found := t.usecase.Feed(t.feed.APIID, t.user)
	t.True(found)
}

func (t *FeedsSuite) TestEditFeed() {
	newFeed, err := t.usecase.Edit(t.feed.APIID, models.Feed{Title: "New Title"}, t.user)
	t.NoError(err)
	t.Equal("New Title", newFeed.Title)
}

func (t *FeedsSuite) TestEditMissingFeed() {
	_, err := t.usecase.Edit("bogus", models.Feed{}, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestDeleteFeed() {
	err := t.usecase.Delete(t.feed.APIID, t.user)
	t.NoError(err)

	_, found := database.FeedWithAPIID(t.feed.APIID, t.user)
	t.False(found)
}

func (t *FeedsSuite) TestDeleteMissingFeed() {
	err := t.usecase.Delete("bogus", t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestMarkFeed() {
	_, err := database.NewEntry(models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, t.feed.APIID, t.user)
	t.Require().NoError(err)

	t.Require().Empty(database.FeedEntries(t.feed.APIID, true, models.MarkerRead, t.user))

	t.usecase.Mark(t.feed.APIID, models.MarkerRead, t.user)

	t.NotEmpty(database.FeedEntries(t.feed.APIID, true, models.MarkerRead, t.user))
}

func (t *FeedsSuite) TestMarkMissingFeed() {
	err := t.usecase.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestFeedEntries() {
	entry, err := database.NewEntry(models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, t.feed.APIID, t.user)
	t.Require().NoError(err)

	entries, err := t.usecase.Entries(t.feed.APIID, true, models.MarkerAny, t.user)
	t.NoError(err)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *FeedsSuite) TestMissingFeedEntries() {
	_, err := t.usecase.Entries("bogus", true, models.MarkerAny, t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestFeedStats() {
	_, err := t.usecase.Stats(t.feed.APIID, t.user)
	t.NoError(err)
}

func (t *FeedsSuite) TestMissingFeedStats() {
	_, err := t.usecase.Stats("bogus", t.user)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) SetupTest() {
	t.usecase = new(FeedUsecase)

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

func (t *FeedsSuite) TearDownTest() {
	err := database.Close()
	t.NoError(err)
}

func TestFeeds(t *testing.T) {
	suite.Run(t, new(FeedsSuite))
}
