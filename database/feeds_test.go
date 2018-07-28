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
	"time"

	"github.com/varddum/syndication/models"
)

func (s *DatabaseTestSuite) TestNewFeedWithDefaults() {
	feed := NewFeed("Test site", "http://example.com", s.user)
	s.Require().NotZero(feed.ID)

	query, found := FeedWithAPIID(feed.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.Title)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)

	s.NotZero(query.Category.ID)
	s.NotEmpty(query.Category.APIID)
	s.Equal(query.Category.Name, models.Uncategorized)

	feeds := CategoryFeeds(query.Category.APIID, s.user)
	s.NotEmpty(feeds)
	s.Equal(feeds[0].Title, feed.Title)
	s.Equal(feeds[0].ID, feed.ID)
	s.Equal(feeds[0].APIID, feed.APIID)
}

func (s *DatabaseTestSuite) TestNewFeedWithCategory() {
	ctg := NewCategory("News", s.user)
	s.NotEmpty(ctg.APIID)
	s.NotZero(ctg.ID)
	s.Empty(ctg.Feeds)

	feed, err := NewFeedWithCategory("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().Nil(err)

	query, found := FeedWithAPIID(feed.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.Title)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)

	s.NotZero(query.Category.ID)
	s.NotEmpty(query.Category.APIID)
	s.Equal(query.Category.Name, "News")

	feeds := CategoryFeeds(ctg.APIID, s.user)
	s.NotEmpty(feeds)
	s.Equal(feeds[0].Title, feed.Title)
	s.Equal(feeds[0].ID, feed.ID)
	s.Equal(feeds[0].APIID, feed.APIID)
}

func (s *DatabaseTestSuite) TestNewFeedWithNonExistingCategory() {
	_, err := NewFeedWithCategory("Test site", "http://example.com", createAPIID(), s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestFeedsFromNonExistingCategory() {
	feeds := CategoryFeeds(createAPIID(), s.user)
	s.Empty(feeds)
}

func (s *DatabaseTestSuite) TestChangeFeedCategory() {
	firstCtg := NewCategory("News", s.user)
	secondCtg := NewCategory("Tech", s.user)

	feed, err := NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID, s.user)
	s.Require().Nil(err)

	feeds := CategoryFeeds(firstCtg.APIID, s.user)
	s.Require().Len(feeds, 1)
	s.Equal(feeds[0].APIID, feed.APIID)
	s.Equal(feeds[0].Title, feed.Title)

	feeds = CategoryFeeds(secondCtg.APIID, s.user)
	s.Empty(feeds)

	err = ChangeFeedCategory(feed.APIID, secondCtg.APIID, s.user)
	s.Nil(err)

	feeds = CategoryFeeds(firstCtg.APIID, s.user)
	s.Empty(feeds)

	feeds = CategoryFeeds(secondCtg.APIID, s.user)
	s.Require().Len(feeds, 1)
	s.Equal(feeds[0].APIID, feed.APIID)
	s.Equal(feeds[0].Title, feed.Title)
}

func (s *DatabaseTestSuite) TestChangeUnknownFeedCategory() {
	err := ChangeFeedCategory("bogus", "none", s.user)
	s.NotNil(err)
}

func (s *DatabaseTestSuite) TestChangeFeedCategoryToUnknown() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	err := ChangeFeedCategory(feed.APIID, "bogus", s.user)
	s.NotNil(err)
}

func (s *DatabaseTestSuite) TestFeeds() {
	for i := 0; i < 5; i++ {
		feed := NewFeed("Test site "+strconv.Itoa(i), "http://example.com", s.user)
		s.Require().NotZero(feed.ID)
		s.Require().NotEmpty(feed.APIID)
	}

	feeds := Feeds(s.user)
	s.Len(feeds, 5)
}

func (s *DatabaseTestSuite) TestEditFeed() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	mdfFeed, err := EditFeed(feed.APIID, models.Feed{Title: "Testing New Name", Subscription: "http://example.com/feed"}, s.user)
	s.Nil(err)

	s.NotEqual(feed.Title, mdfFeed.Title)
	s.Equal("Testing New Name", mdfFeed.Title)

	s.NotEqual(feed.Subscription, mdfFeed.Subscription)
	s.Equal("http://example.com/feed", mdfFeed.Subscription)
}

func (s *DatabaseTestSuite) TestEditNonExistingFeed() {
	_, err := EditFeed("", models.Feed{}, s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestDeleteFeed() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	query, found := FeedWithAPIID(feed.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.APIID)

	err := DeleteFeed(feed.APIID, s.user)
	s.Nil(err)

	_, found = FeedWithAPIID(feed.APIID, s.user)
	s.False(found)
}

func (s *DatabaseTestSuite) TestDeleteNonExistingFeed() {
	err := DeleteFeed(createAPIID(), s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestFeedEntriesFromNonExistenFeed() {
	entries := FeedEntries(createAPIID(), true, models.MarkerUnread, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestMarkFeed() {
	firstFeed := NewFeed("Test site", "http://example.com", s.user)
	s.NotEmpty(firstFeed.APIID)

	secondFeed := NewFeed("Test site", "http://example.com", s.user)
	s.NotEmpty(secondFeed.APIID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:     "First Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.MarkerUnread,
				Published: time.Now(),
			}

			_, err := NewEntry(entry, firstFeed.APIID, s.user)
			s.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.MarkerRead,
				Published: time.Now(),
			}

			_, err := NewEntry(entry, secondFeed.APIID, s.user)
			s.Require().Nil(err)
		}

	}

	s.Require().Equal(defaultInstance.db.Model(&s.user).Association("Entries").Count(), 10)
	s.Require().Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerRead).Association("Entries").Count(), 5)
	s.Require().Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count(), 5)

	err := MarkFeed(firstFeed.APIID, models.MarkerRead, s.user)
	s.Nil(err)

	entries := FeedEntries(firstFeed.APIID, true, models.MarkerRead, s.user)
	s.Len(entries, 5)

	s.Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerRead).Association("Entries").Count(), 10)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.MarkerRead)
	}

	err = MarkFeed(secondFeed.APIID, models.MarkerUnread, s.user)
	s.Nil(err)

	s.Equal(defaultInstance.db.Model(&s.user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count(), 5)

	entries = FeedEntries(secondFeed.APIID, true, models.MarkerUnread, s.user)
	s.Nil(err)
	s.Len(entries, 5)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.MarkerUnread)
	}
}

func (s *DatabaseTestSuite) TestFeedStats() {
	feed := NewFeed("News", "http://example.com", s.user)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerRead,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := NewEntry(entry, feed.APIID, s.user)
		s.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err := NewEntry(entry, feed.APIID, s.user)

		s.Require().Nil(err)
	}

	stats := FeedStats(feed.APIID, s.user)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}
