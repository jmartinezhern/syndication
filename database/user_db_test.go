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
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/models"
)

type (
	UserDBTestSuite struct {
		suite.Suite

		db   UserDB
		gDB  *DB
		user models.User
	}
)

const TestUserDBPath = "/tmp/syndication-test-db.db"

func (s *UserDBTestSuite) SetupTest() {
	var err error
	s.gDB, err = NewDB(config.Database{
		Connection: TestUserDBPath,
		Type:       "sqlite3",
	})
	s.Require().NotNil(s.gDB)
	s.Require().Nil(err)

	user := s.gDB.NewUser("test", "golang")
	s.Require().NotZero(user.ID)

	s.db = s.gDB.NewUserDB(user)
}

func (s *UserDBTestSuite) TearDownTest() {
	err := s.gDB.Close()
	s.Nil(err)
	err = os.Remove(s.gDB.config.Connection)
	s.Nil(err)
}

func (s *UserDBTestSuite) TestNewCategory() {
	ctg := s.db.NewCategory("News")
	s.NotEmpty(ctg.APIID)
	s.NotZero(ctg.ID)
	s.NotZero(ctg.UserID)
	s.NotZero(ctg.CreatedAt)
	s.NotZero(ctg.UpdatedAt)
	s.NotZero(ctg.UserID)

	query, found := s.db.CategoryWithAPIID(ctg.APIID)
	s.True(found)
	s.NotEmpty(query.Name)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)
}

func (s *UserDBTestSuite) TestCategories() {
	for i := 0; i < 5; i++ {
		ctg := s.db.NewCategory("Test Category " + strconv.Itoa(i))
		s.Require().NotZero(ctg.ID)
	}

	ctgs := s.db.Categories()
	s.Len(ctgs, 6)
}

func (s *UserDBTestSuite) TestEditCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotZero(ctg.ID)

	query, found := s.db.CategoryWithAPIID(ctg.APIID)
	s.True(found)
	s.Equal(query.Name, "News")

	ctg.Name = "World News"
	err := s.db.EditCategory(&ctg)
	s.Require().Nil(err)

	query, found = s.db.CategoryWithAPIID(ctg.APIID)
	s.True(found)
	s.Equal(ctg.ID, query.ID)
	s.Equal(query.Name, "World News")
}

func (s *UserDBTestSuite) TestEditNonExistingCategory() {
	err := s.db.EditCategory(&models.Category{})
	s.Equal(ErrModelNotFound, err)
}

func (s *UserDBTestSuite) TestDeleteCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotZero(ctg.ID)

	query, found := s.db.CategoryWithAPIID(ctg.APIID)
	s.True(found)
	s.NotEmpty(query.APIID)

	err := s.db.DeleteCategory(ctg.APIID)
	s.Nil(err)

	_, found = s.db.CategoryWithAPIID(ctg.APIID)
	s.False(found)
}

func (s *UserDBTestSuite) TestDeleteNonExistingCategory() {
	err := s.db.DeleteCategory(createAPIID())
	s.Equal(ErrModelNotFound, err)
}

func (s *UserDBTestSuite) TestNewFeedWithDefaults() {
	feed := s.db.NewFeed("Test site", "http://example.com")
	s.Require().NotZero(feed.ID)

	query, found := s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
	s.NotEmpty(query.Title)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)

	s.NotZero(query.Category.ID)
	s.NotEmpty(query.Category.APIID)
	s.Equal(query.Category.Name, models.Uncategorized)

	feeds := s.db.FeedsFromCategory(query.Category.APIID)
	s.NotEmpty(feeds)
	s.Equal(feeds[0].Title, feed.Title)
	s.Equal(feeds[0].ID, feed.ID)
	s.Equal(feeds[0].APIID, feed.APIID)
}

func (s *UserDBTestSuite) TestNewFeedWithCategory() {
	ctg := s.db.NewCategory("News")
	s.NotEmpty(ctg.APIID)
	s.NotZero(ctg.ID)
	s.Empty(ctg.Feeds)

	feed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", ctg.APIID)
	s.Require().Nil(err)

	query, found := s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
	s.NotEmpty(query.Title)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)

	s.NotZero(query.Category.ID)
	s.NotEmpty(query.Category.APIID)
	s.Equal(query.Category.Name, "News")

	feeds := s.db.FeedsFromCategory(ctg.APIID)
	s.NotEmpty(feeds)
	s.Equal(feeds[0].Title, feed.Title)
	s.Equal(feeds[0].ID, feed.ID)
	s.Equal(feeds[0].APIID, feed.APIID)
}

func (s *UserDBTestSuite) TestNewFeedWithNonExistingCategory() {
	_, err := s.db.NewFeedWithCategory("Test site", "http://example.com", createAPIID())
	s.Equal(ErrModelNotFound, err)
}

func (s *UserDBTestSuite) TestFeedsFromNonExistingCategory() {
	feeds := s.db.FeedsFromCategory(createAPIID())
	s.Empty(feeds)
}

func (s *UserDBTestSuite) TestChangeFeedCategory() {
	firstCtg := s.db.NewCategory("News")
	secondCtg := s.db.NewCategory("Tech")

	feed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID)
	s.Require().Nil(err)

	feeds := s.db.FeedsFromCategory(firstCtg.APIID)
	s.Require().Len(feeds, 1)
	s.Equal(feeds[0].APIID, feed.APIID)
	s.Equal(feeds[0].Title, feed.Title)

	feeds = s.db.FeedsFromCategory(secondCtg.APIID)
	s.Empty(feeds)

	err = s.db.ChangeFeedCategory(feed.APIID, secondCtg.APIID)
	s.Nil(err)

	feeds = s.db.FeedsFromCategory(firstCtg.APIID)
	s.Empty(feeds)

	feeds = s.db.FeedsFromCategory(secondCtg.APIID)
	s.Require().Len(feeds, 1)
	s.Equal(feeds[0].APIID, feed.APIID)
	s.Equal(feeds[0].Title, feed.Title)
}

func (s *UserDBTestSuite) TestChangeUnknownFeedCategory() {
	err := s.db.ChangeFeedCategory("bogus", "none")
	s.NotNil(err)
}

func (s *UserDBTestSuite) TestChangeFeedCategoryToUnknown() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	err := s.db.ChangeFeedCategory(feed.APIID, "bogus")
	s.NotNil(err)
}

func (s *UserDBTestSuite) TestFeeds() {
	for i := 0; i < 5; i++ {
		feed := s.db.NewFeed("Test site "+strconv.Itoa(i), "http://example.com")
		s.Require().NotZero(feed.ID)
		s.Require().NotEmpty(feed.APIID)
	}

	feeds := s.db.Feeds()
	s.Len(feeds, 5)
}

func (s *UserDBTestSuite) TestEditFeed() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	query, found := s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
	s.NotEmpty(query.Title)
	s.NotZero(query.ID)

	feed.Title = "Testing New Name"
	feed.Subscription = "http://example.com/feed"

	err := s.db.EditFeed(&feed)
	s.Nil(err)

	query, found = s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
	s.Equal(feed.Title, "Testing New Name")
	s.Equal(feed.Subscription, "http://example.com/feed")
}

func (s *UserDBTestSuite) TestEditNonExistingFeed() {
	err := s.db.EditFeed(&models.Feed{})
	s.Equal(ErrModelNotFound, err)
}

func (s *UserDBTestSuite) TestDeleteFeed() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	query, found := s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
	s.NotEmpty(query.APIID)

	err := s.db.DeleteFeed(feed.APIID)
	s.Nil(err)

	_, found = s.db.FeedWithAPIID(feed.APIID)
	s.False(found)
}

func (s *UserDBTestSuite) TestDeleteNonExistingFeed() {
	err := s.db.DeleteFeed(createAPIID())
	s.Equal(ErrModelNotFound, err)
}

func (s *UserDBTestSuite) TestNewTag() {
	tag := s.db.NewTag("tech")

	query, found := s.db.TagWithAPIID(tag.APIID)
	s.True(found)
	s.NotEmpty(query.Name)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)
}

func (s *UserDBTestSuite) TestTagEntries() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.Unread,
		}

		entries = append(entries, entry)
	}

	entries, err := s.db.NewEntries(entries, feed.APIID)
	s.Require().NotEmpty(entries)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Any)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	tag := s.db.NewTag("Tech")
	s.Nil(err)
	s.NotZero(tag.ID)

	dbTag, found := s.db.TagWithAPIID(tag.APIID)
	s.True(found)
	s.Equal(tag.APIID, dbTag.APIID)
	s.Equal(tag.Name, dbTag.Name)
	s.NotZero(dbTag.ID)

	err = s.db.TagEntries(tag.APIID, entryAPIIDs)
	s.Nil(err)

	taggedEntries := s.db.EntriesFromTag(tag.APIID, models.Any, true)
	s.Nil(err)
	s.Len(taggedEntries, 5)
}

func (s *UserDBTestSuite) TestTagMultipleEntries() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry " + strconv.Itoa(i),
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.Unread,
		}

		entries = append(entries, entry)
	}

	_, err := s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	secondTag := models.Tag{
		Name: "Second tag",
	}

	firstTag := s.db.NewTag("First tag")
	s.NotZero(firstTag.ID)

	secondTag = s.db.NewTag("Second Tag")
	s.NotZero(secondTag.ID)

	s.Require().NotEqual(firstTag.APIID, secondTag.APIID)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Any)
	s.Require().Nil(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = s.db.TagEntries(firstTag.APIID, entryAPIIDs)
	s.Nil(err)

	err = s.db.TagEntries(secondTag.APIID, entryAPIIDs)
	s.Nil(err)

	taggedEntries := s.db.EntriesFromTag(firstTag.APIID, models.Any, true)
	s.Len(taggedEntries, 5)

	taggedEntries = s.db.EntriesFromTag(secondTag.APIID, models.Any, true)
	s.Len(taggedEntries, 5)
}

func (s *UserDBTestSuite) TestEntriesFromMultipleTags() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	var entries []models.Entry
	for i := 0; i < 15; i++ {
		entry := models.Entry{
			Title:  "Test Entry " + strconv.Itoa(i),
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.Unread,
		}

		entries = append(entries, entry)
	}

	entries, err := s.db.NewEntries(entries, feed.APIID)
	s.Require().Len(entries, 15)
	s.Require().Nil(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		s.Require().NotEmpty(entry.APIID)
		entryAPIIDs[i] = entry.APIID
	}

	firstTag := s.db.NewTag("First tag")
	s.NotZero(firstTag.ID)

	secondTag := s.db.NewTag("Second tag")
	s.NotZero(secondTag.ID)

	err = s.db.TagEntries(firstTag.APIID, entryAPIIDs[0:5])
	s.Nil(err)

	err = s.db.TagEntries(secondTag.APIID, entryAPIIDs[5:10])
	s.Nil(err)

	s.Require().NotEqual(firstTag.APIID, secondTag.APIID)

	taggedEntries := s.db.EntriesFromMultipleTags([]string{firstTag.APIID, secondTag.APIID}, true, models.Any)
	s.Len(taggedEntries, 10)
}

func (s *UserDBTestSuite) TestDeleteTag() {
	tag := s.db.NewTag("News")

	query, found := s.db.TagWithAPIID(tag.APIID)
	s.True(found)
	s.NotEmpty(query.APIID)

	err := s.db.DeleteTag(tag.APIID)
	s.Nil(err)

	_, found = s.db.TagWithAPIID(tag.APIID)
	s.False(found)
}

func (s *UserDBTestSuite) TestEditTag() {
	tag := s.db.NewTag("News")

	query, found := s.db.TagWithAPIID(tag.APIID)
	s.True(found)
	s.Equal(query.Name, "News")

	err := s.db.EditTagName(tag.APIID, "World News")
	s.Require().Nil(err)

	query, found = s.db.TagWithAPIID(tag.APIID)
	s.True(found)
	s.Equal(tag.ID, query.ID)
	s.Equal(query.Name, "World News")
}

func (s *UserDBTestSuite) TestNewEntry() {
	feed := s.db.NewFeed("Test site", "http://example.com")
	s.NotZero(feed.ID)
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, feed.APIID)
	s.Require().Nil(err)
	s.NotEmpty(entry.APIID)

	query, found := s.db.EntryWithAPIID(entry.APIID)
	s.True(found)
	s.NotZero(query.FeedID)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Unread)
	s.NotEmpty(entries)
	s.Len(entries, 1)
	s.Equal(entries[0].Title, entry.Title)
}

func (s *UserDBTestSuite) TestEntriesFromFeedWithNonExistenFeed() {
	entries := s.db.EntriesFromFeed(createAPIID(), true, models.Unread)
	s.Empty(entries)
}

func (s *UserDBTestSuite) TestNewEntryWithEmptyFeed() {
	entry := models.Entry{
		Title:     "Test Entry",
		Link:      "http://example.com",
		Author:    "varddum",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, createAPIID())
	s.NotNil(err)
	s.Equal(ErrModelNotFound, err)
	s.Zero(entry.ID)
	s.Empty(entry.APIID)

	query, found := s.db.EntryWithAPIID(entry.APIID)
	s.False(found)
	s.Zero(query.FeedID)
}

func (s *UserDBTestSuite) TestNewEntryWithBadFeed() {
	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, createAPIID())
	s.NotNil(err)
	s.Zero(entry.ID)
	s.Empty(entry.APIID)

	query, found := s.db.EntryWithAPIID(entry.APIID)
	s.False(found)
	s.Zero(query.FeedID)
}

func (s *UserDBTestSuite) TestNewEntries() {
	feed := s.db.NewFeed("Test site", "http://example.com")
	s.NotZero(feed.ID)
	s.NotEmpty(feed.APIID)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:     "Test Entry",
			Author:    "varddum",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		entries = append(entries, entry)
	}

	entries, err := s.db.NewEntries(entries, feed.APIID)
	s.Require().Len(entries, 5)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Unread)
	s.Len(entries, 5)
	for _, entry := range entries {
		s.NotZero(entry.ID)
		s.NotZero(entry.Title)
	}
}

func (s *UserDBTestSuite) TestNewEntriesWithBadFeed() {
	entries := []models.Entry{
		{},
	}
	_, err := s.db.NewEntries(entries, "")
	s.NotNil(err)
}

func (s *UserDBTestSuite) TestEntries() {
	feed := s.db.NewFeed("Test site", "http://example.com")
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, feed.APIID)
	s.Require().Nil(err)

	entries := s.db.Entries(true, models.Unread)
	s.NotEmpty(entries)
	s.Equal(entries[0].Title, entry.Title)
}

func (s *UserDBTestSuite) TestEntriesWithNoneMarker() {
	entries := s.db.Entries(true, models.None)
	s.Empty(entries)
}

func (s *UserDBTestSuite) TestEntriesFromFeed() {
	feed := s.db.NewFeed("Test site", "http://example.com")
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, feed.APIID)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Unread)
	s.Require().NotEmpty(entries)
	s.Equal(entries[0].Title, entry.Title)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Read)
	s.Empty(entries)
}

func (s *UserDBTestSuite) TestEntriesFromFeedWithNoneMarker() {
	entries := s.db.EntriesFromFeed("bogus", true, models.None)
	s.Empty(entries)
}

func (s *UserDBTestSuite) TestEntryWithGUIDExists() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	entry := models.Entry{
		Title:     "Test Entry",
		GUID:      "entry@test",
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, feed.APIID)
	s.Nil(err)
	s.True(s.db.EntryWithGUIDExists(entry.GUID, feed.APIID))
}

func (s *UserDBTestSuite) TestEntryWithGUIDDoesNotExists() {
	feed := s.db.NewFeed("Test site", "http://example.com")

	entry := models.Entry{
		Title:     "Test Entry",
		Published: time.Now(),
	}

	_, err := s.db.NewEntry(entry, feed.APIID)
	s.Nil(err)
	s.False(s.db.EntryWithGUIDExists("item@test", feed.APIID))
}

func (s *UserDBTestSuite) TestEntriesFromCategory() {
	firstCtg := s.db.NewCategory("News")
	s.NotEmpty(firstCtg.APIID)

	secondCtg := s.db.NewCategory("Tech")
	s.NotEmpty(secondCtg.APIID)

	firstFeed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID)
	s.Nil(err)

	secondFeed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID)
	s.Nil(err)

	thirdFeed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID)
	s.Nil(err)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:     "First Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err = s.db.NewEntry(entry, firstFeed.APIID)
			s.Require().Nil(err)
		} else if i < 7 {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err = s.db.NewEntry(entry, secondFeed.APIID)
			s.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Third Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err = s.db.NewEntry(entry, thirdFeed.APIID)
			s.Require().Nil(err)
		}

	}

	entries := s.db.EntriesFromCategory(firstCtg.APIID, false, models.Unread)
	s.NotEmpty(entries)
	s.Len(entries, 5)
	s.Equal(entries[0].Title, "First Feed Test Entry 0")

	entries = s.db.EntriesFromCategory(secondCtg.APIID, true, models.Unread)
	s.NotEmpty(entries)
	s.Len(entries, 5)
	s.Equal(entries[0].Title, "Third Feed Test Entry 9")
	s.Equal(entries[len(entries)-1].Title, "Second Feed Test Entry 5")
}

func (s *UserDBTestSuite) TestEntriesFromCategoryWithtNoneMarker() {
	entries := s.db.EntriesFromCategory("bogus", true, models.None)
	s.Empty(entries)
}

func (s *UserDBTestSuite) TestEntriesFromNonExistingCategory() {
	entries := s.db.EntriesFromCategory(createAPIID(), true, models.Unread)
	s.Empty(entries)
}

func (s *UserDBTestSuite) TestMarkCategory() {
	firstCtg := s.db.NewCategory("News")
	s.NotEmpty(firstCtg.APIID)

	secondCtg := s.db.NewCategory("Tech")
	s.NotEmpty(secondCtg.APIID)

	firstFeed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID)
	s.Require().Nil(err)
	s.NotEmpty(firstFeed.APIID)

	secondFeed, err := s.db.NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID)
	s.Require().Nil(err)
	s.NotEmpty(firstFeed.APIID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:     "First Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err = s.db.NewEntry(entry, firstFeed.APIID)
			s.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Read,
				Published: time.Now(),
			}

			_, err = s.db.NewEntry(entry, secondFeed.APIID)
			s.Require().Nil(err)
		}

	}

	s.Require().Equal(s.db.db.Model(&s.db.user).Association("Entries").Count(), 10)
	s.Require().Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	s.Require().Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err = s.db.MarkCategory(firstCtg.APIID, models.Read)
	s.Nil(err)

	entries := s.db.EntriesFromCategory(firstCtg.APIID, true, models.Any)
	s.Len(entries, 5)

	s.Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.Read)
	}

	err = s.db.MarkCategory(secondCtg.APIID, models.Unread)
	s.Nil(err)

	s.Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries = s.db.EntriesFromCategory(secondCtg.APIID, true, models.Any)
	s.Len(entries, 5)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.Unread)
	}
}

func (s *UserDBTestSuite) TestMarkFeed() {
	firstFeed := s.db.NewFeed("Test site", "http://example.com")
	s.NotEmpty(firstFeed.APIID)

	secondFeed := s.db.NewFeed("Test site", "http://example.com")
	s.NotEmpty(secondFeed.APIID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:     "First Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err := s.db.NewEntry(entry, firstFeed.APIID)
			s.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Read,
				Published: time.Now(),
			}

			_, err := s.db.NewEntry(entry, secondFeed.APIID)
			s.Require().Nil(err)
		}

	}

	s.Require().Equal(s.db.db.Model(&s.db.user).Association("Entries").Count(), 10)
	s.Require().Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	s.Require().Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err := s.db.MarkFeed(firstFeed.APIID, models.Read)
	s.Nil(err)

	entries := s.db.EntriesFromFeed(firstFeed.APIID, true, models.Read)
	s.Len(entries, 5)

	s.Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.Read)
	}

	err = s.db.MarkFeed(secondFeed.APIID, models.Unread)
	s.Nil(err)

	s.Equal(s.db.db.Model(&s.db.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries = s.db.EntriesFromFeed(secondFeed.APIID, true, models.Unread)
	s.Nil(err)
	s.Len(entries, 5)

	for _, entry := range entries {
		s.EqualValues(entry.Mark, models.Unread)
	}
}

func (s *UserDBTestSuite) TestMarkEntry() {
	feed := s.db.NewFeed("News", "http://localhost/news")

	entry := models.Entry{
		Title:     "Article",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := s.db.NewEntry(entry, feed.APIID)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Unread)
	s.Require().Len(entries, 1)

	err = s.db.MarkEntry(entry.APIID, models.Read)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Unread)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Read)
	s.Require().Len(entries, 1)
}

func (s *UserDBTestSuite) TestStats() {
	feed := s.db.NewFeed("News", "http://example.com")
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := s.db.NewEntry(entry, feed.APIID)
		s.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		_, err := s.db.NewEntry(entry, feed.APIID)
		s.Require().Nil(err)
	}

	stats := s.db.Stats()
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *UserDBTestSuite) TestFeedStats() {
	feed := s.db.NewFeed("News", "http://example.com")
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := s.db.NewEntry(entry, feed.APIID)
		s.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		_, err := s.db.NewEntry(entry, feed.APIID)
		s.Require().Nil(err)
	}

	stats := s.db.FeedStats(feed.APIID)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *UserDBTestSuite) TestCategoryStats() {
	ctg := s.db.NewCategory("World")
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("News", "http://example.com", ctg.APIID)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		_, err = s.db.NewEntry(entry, feed.APIID)
		s.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		_, err = s.db.NewEntry(entry, feed.APIID)
		s.Require().Nil(err)
	}

	stats := s.db.CategoryStats(ctg.APIID)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *UserDBTestSuite) TestKeyBelongsToUser() {
	key, err := s.db.NewAPIKey("secret", time.Hour)
	s.Require().Nil(err)

	found := s.db.KeyBelongsToUser(models.APIKey{Key: key.Key})
	s.True(found)
}

func (s *UserDBTestSuite) TestKeyDoesNotBelongToUser() {
	key := models.APIKey{
		Key: "123456789",
	}

	found := s.db.KeyBelongsToUser(key)
	s.False(found)
}

func TestUserDBTestSuite(t *testing.T) {
	suite.Run(t, new(UserDBTestSuite))
}
