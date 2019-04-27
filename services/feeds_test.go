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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type FeedsSuite struct {
	suite.Suite

	service Feed

	db          *sql.DB
	feedsRepo   repo.Feeds
	entriesRepo repo.Entries
	ctgsRepo    repo.Categories

	user *models.User
	feed models.Feed
}

func (t *FeedsSuite) TestNewFeed() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintln(w, "<rss></rss>")
		t.Require().NoError(err)
	}))
	defer ts.Close()

	feed, err := t.service.New("Example", ts.URL, "", t.user.ID)
	t.NoError(err)
	_, found := t.feedsRepo.FeedWithID(t.user.ID, feed.ID)
	t.True(found)
}

func (t *FeedsSuite) TestUnreachableNewFeed() {
	_, err := t.service.New("Example", "bogus", "", t.user.ID)
	t.EqualError(err, ErrFetchingFeed.Error())
}

func (t *FeedsSuite) TestFeeds() {
	feeds, _ := t.service.Feeds("", 2, t.user.ID)
	t.Len(feeds, 1)
	t.Equal(t.feed.Title, feeds[0].Title)
}

func (t *FeedsSuite) TestFeed() {
	_, found := t.service.Feed(t.feed.ID, t.user.ID)
	t.True(found)
}

func (t *FeedsSuite) TestEditFeed() {
	feed := models.Feed{ID: t.feed.ID, Title: "New Title"}
	err := t.service.Update(&feed, t.user.ID)
	t.NoError(err)

	updatedFeed, _ := t.feedsRepo.FeedWithID(t.user.ID, t.feed.ID)
	t.Equal("New Title", updatedFeed.Title)
}

func (t *FeedsSuite) TestEditMissingFeed() {
	err := t.service.Update(&models.Feed{}, t.user.ID)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestDeleteFeed() {
	err := t.service.Delete(t.feed.ID, t.user.ID)
	t.NoError(err)

	_, found := t.feedsRepo.FeedWithID(t.user.ID, t.feed.ID)
	t.False(found)
}

func (t *FeedsSuite) TestDeleteMissingFeed() {
	err := t.service.Delete("bogus", t.user.ID)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestMarkFeed() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	err := t.service.Mark(t.feed.ID, models.MarkerRead, t.user.ID)
	t.NoError(err)

	entries, _ := sql.NewEntries(t.db).ListFromFeed(t.user.ID, t.feed.ID, "", 1, false, models.MarkerAny)
	t.Require().Len(entries, 1)
	t.Equal(entry.ID, entries[0].ID)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *FeedsSuite) TestMarkMissingFeed() {
	err := t.service.Mark("bogus", models.MarkerRead, t.user.ID)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) TestFeedEntries() {
	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  t.feed,
	}
	t.entriesRepo.Create(t.user.ID, &entry)

	entries, _ := t.service.Entries(t.feed.ID, "", 1, true, models.MarkerAny, t.user.ID)
	t.Len(entries, 1)
	t.Equal(entry.Title, entries[0].Title)
}

func (t *FeedsSuite) TestMissingFeedEntries() {
	entries, _ := t.service.Entries(t.feed.ID, "", 1, true, models.MarkerAny, t.user.ID)
	t.Len(entries, 0)
}

func (t *FeedsSuite) TestFeedStats() {
	_, err := t.service.Stats(t.feed.ID, t.user.ID)
	t.NoError(err)
}

func (t *FeedsSuite) TestMissingFeedStats() {
	_, err := t.service.Stats("bogus", t.user.ID)
	t.EqualError(err, ErrFeedNotFound.Error())
}

func (t *FeedsSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")
	t.feedsRepo = sql.NewFeeds(t.db)
	t.entriesRepo = sql.NewEntries(t.db)
	t.ctgsRepo = sql.NewCategories(t.db)

	t.service = NewFeedsService(t.feedsRepo, t.ctgsRepo, t.entriesRepo)

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
	t.feedsRepo.Create(t.user.ID, &t.feed)
}

func (t *FeedsSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestFeeds(t *testing.T) {
	suite.Run(t, new(FeedsSuite))
}
