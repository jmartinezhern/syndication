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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type FeedsSuite struct {
	suite.Suite

	user models.User
	ctg  models.Category
}

func (s *FeedsSuite) TestNewFeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	query, found := FeedWithAPIID(feed.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.Title)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)

	s.NotZero(query.Category.ID)
	s.NotEmpty(query.Category.APIID)
	s.Equal(query.Category.Name, s.ctg.Name)

	feeds := CategoryFeeds(s.ctg.APIID, s.user)
	s.NotEmpty(feeds)
	s.Equal(feeds[0].Title, feed.Title)
	s.Equal(feeds[0].ID, feed.ID)
	s.Equal(feeds[0].APIID, feed.APIID)
}

func (s *FeedsSuite) TestNewFeedWithNonExistingCategory() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, "bogus", s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *FeedsSuite) TestFeedsFromNonExistingCategory() {
	feeds := CategoryFeeds("bogus", s.user)
	s.Empty(feeds)
}

func (s *FeedsSuite) TestChangeFeedCategory() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "tech",
	}, s.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	feeds := CategoryFeeds(s.ctg.APIID, s.user)
	s.Require().Len(feeds, 1)
	s.Equal(feeds[0].APIID, feed.APIID)
	s.Equal(feeds[0].Title, feed.Title)

	feeds = CategoryFeeds(ctgID, s.user)
	s.Empty(feeds)

	err = ChangeFeedCategory(feed.APIID, ctgID, s.user)
	s.NoError(err)

	feeds = CategoryFeeds(s.ctg.APIID, s.user)
	s.Empty(feeds)

	feeds = CategoryFeeds(ctgID, s.user)
	s.Require().Len(feeds, 1)
	s.Equal(feeds[0].APIID, feed.APIID)
	s.Equal(feeds[0].Title, feed.Title)
}

func (s *FeedsSuite) TestChangeUnknownFeedCategory() {
	err := ChangeFeedCategory("bogus", "none", s.user)
	s.Error(err)
}

func (s *FeedsSuite) TestChangeFeedCategoryToUnknown() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	err = ChangeFeedCategory(feed.APIID, "bogus", s.user)
	s.Error(err)
}

func (s *FeedsSuite) TestFeeds() {
	var feeds []models.Feed
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			APIID:        utils.CreateAPIID(),
			Title:        "Test site " + strconv.Itoa(i),
			Subscription: "http://example.com",
		}
		err := CreateFeed(&feed, s.ctg.APIID, s.user)
		s.Require().NoError(err)

		feeds = append(feeds, feed)
	}

	cFeeds, continuationID := Feeds("", 2, s.user)
	s.Equal(feeds[2].APIID, continuationID)
	s.Require().Len(cFeeds, 2)
	s.Equal(feeds[0].Title, cFeeds[0].Title)
	s.Equal(feeds[1].Title, cFeeds[1].Title)

	cFeeds, continuationID = Feeds(continuationID, 3, s.user)
	s.Len(continuationID, 0)
	s.Require().Len(cFeeds, 3)
	s.Equal(feeds[2].Title, cFeeds[0].Title)
	s.Equal(feeds[3].Title, cFeeds[1].Title)
	s.Equal(feeds[4].Title, cFeeds[2].Title)
}

func (s *FeedsSuite) TestEditFeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	mdfFeed, err := EditFeed(feed.APIID, models.Feed{Title: "Testing New Name", Subscription: "http://example.com/feed"}, s.user)
	s.NoError(err)

	s.NotEqual(feed.Title, mdfFeed.Title)
	s.Equal("Testing New Name", mdfFeed.Title)

	s.NotEqual(feed.Subscription, mdfFeed.Subscription)
	s.Equal("http://example.com/feed", mdfFeed.Subscription)
}

func (s *FeedsSuite) TestEditNonExistingFeed() {
	_, err := EditFeed("", models.Feed{}, s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *FeedsSuite) TestDeleteFeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	query, found := FeedWithAPIID(feed.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.APIID)

	err = DeleteFeed(feed.APIID, s.user)
	s.NoError(err)

	_, found = FeedWithAPIID(feed.APIID, s.user)
	s.False(found)
}

func (s *FeedsSuite) TestDeleteNonExistingFeed() {
	err := DeleteFeed("bogus", s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *FeedsSuite) TestFeedEntriesFromNonExistenFeed() {
	entries := FeedEntries("bogus", true, models.MarkerUnread, s.user)
	s.Empty(entries)
}

func (s *FeedsSuite) TestMarkFeed() {
	firstFeed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&firstFeed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	secondFeed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err = CreateFeed(&secondFeed, s.ctg.APIID, s.user)
	s.Require().NoError(err)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:     "First Feed Test Entry " + strconv.Itoa(i),
				Author:    "John Doe",
				Link:      "http://example.com",
				Mark:      models.MarkerUnread,
				Published: time.Now(),
			}

			_, err := NewEntry(entry, firstFeed.APIID, s.user)
			s.Require().NoError(err)
		} else {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "John Doe",
				Link:      "http://example.com",
				Mark:      models.MarkerRead,
				Published: time.Now(),
			}

			_, err := NewEntry(entry, secondFeed.APIID, s.user)
			s.Require().NoError(err)
		}

	}

	s.Require().Equal(defaultInstance.db.Model(&s.user).Association("Entries").Count(), 10)
	s.Require().Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerRead).Association("Entries").Count(), 5)
	s.Require().Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count(), 5)

	err = MarkFeed(firstFeed.APIID, models.MarkerRead, s.user)
	s.NoError(err)

	entries := FeedEntries(firstFeed.APIID, true, models.MarkerRead, s.user)
	s.Len(entries, 5)

	s.Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerRead).Association("Entries").Count(), 10)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.MarkerRead)
	}

	err = MarkFeed(secondFeed.APIID, models.MarkerUnread, s.user)
	s.NoError(err)

	s.Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count(), 5)

	entries = FeedEntries(secondFeed.APIID, true, models.MarkerUnread, s.user)
	s.NoError(err)
	s.Len(entries, 5)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.MarkerUnread)
	}
}

func (s *FeedsSuite) TestFeedStats() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err := CreateFeed(&feed, s.ctg.APIID, s.user)
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

	stats := FeedStats(feed.APIID, s.user)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *FeedsSuite) SetupTest() {
	err := Init("sqlite3", ":memory:")

	s.Require().NoError(err)

	s.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test",
	}

	CreateUser(&s.user)
	s.ctg = models.Category{
		APIID: utils.CreateAPIID(),
		Name:  models.Uncategorized,
	}
	CreateCategory(&s.ctg, s.user)
}

func (s *FeedsSuite) TearDownTest() {
	err := Close()
	s.NoError(err)
}

func TestFeedsSuite(t *testing.T) {
	suite.Run(t, new(FeedsSuite))
}
