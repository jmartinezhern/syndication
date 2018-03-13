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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/models"
)

type (
	DatabaseTestSuite struct {
		suite.Suite

		db   *DB
		user models.User
	}
)

const TestDatabasePath = "/tmp/syndication-test-db.db"

func (suite *DatabaseTestSuite) SetupTest() {
	var err error
	suite.db, err = NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	suite.Require().NotNil(suite.db)
	suite.Require().Nil(err)

	suite.user = suite.db.NewUser("test", "golang")
	suite.Require().NotZero(suite.user.ID)
}

func (suite *DatabaseTestSuite) TearDownTest() {
	err := suite.db.Close()
	suite.Nil(err)
	err = os.Remove(suite.db.config.Connection)
	suite.Nil(err)
}

func (suite *DatabaseTestSuite) TestNewCategory() {
	ctg := suite.db.NewCategory("News", &suite.user)
	suite.NotEmpty(ctg.APIID)
	suite.NotZero(ctg.ID)
	suite.NotZero(ctg.UserID)
	suite.NotZero(ctg.CreatedAt)
	suite.NotZero(ctg.UpdatedAt)
	suite.NotZero(ctg.UserID)

	query, found := suite.db.CategoryWithAPIID(ctg.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.Name)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)
}

func (suite *DatabaseTestSuite) TestCategories() {
	for i := 0; i < 5; i++ {
		ctg := suite.db.NewCategory("Test Category "+strconv.Itoa(i), &suite.user)
		suite.Require().NotZero(ctg.ID)
	}

	ctgs := suite.db.Categories(&suite.user)
	suite.Len(ctgs, 6)
}

func (suite *DatabaseTestSuite) TestEditCategory() {
	ctg := suite.db.NewCategory("News", &suite.user)
	suite.Require().NotZero(ctg.ID)

	query, found := suite.db.CategoryWithAPIID(ctg.APIID, &suite.user)
	suite.True(found)
	suite.Equal(query.Name, "News")

	ctg.Name = "World News"
	err := suite.db.EditCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	query, found = suite.db.CategoryWithAPIID(ctg.APIID, &suite.user)
	suite.True(found)
	suite.Equal(ctg.ID, query.ID)
	suite.Equal(query.Name, "World News")
}

func (suite *DatabaseTestSuite) TestEditNonExistingCategory() {
	err := suite.db.EditCategory(&models.Category{}, &suite.user)
	suite.Equal(ErrModelNotFound, err)
}

func (suite *DatabaseTestSuite) TestDeleteCategory() {
	ctg := suite.db.NewCategory("News", &suite.user)
	suite.Require().NotZero(ctg.ID)

	query, found := suite.db.CategoryWithAPIID(ctg.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.APIID)

	err := suite.db.DeleteCategory(ctg.APIID, &suite.user)
	suite.Nil(err)

	_, found = suite.db.CategoryWithAPIID(ctg.APIID, &suite.user)
	suite.False(found)
}

func (suite *DatabaseTestSuite) TestDeleteNonExistingCategory() {
	err := suite.db.DeleteCategory(createAPIID(), &suite.user)
	suite.Equal(ErrModelNotFound, err)
}

func (suite *DatabaseTestSuite) TestNewFeedWithDefaults() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.Require().NotZero(feed.ID)

	query, found := suite.db.FeedWithAPIID(feed.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)

	suite.NotZero(query.Category.ID)
	suite.NotEmpty(query.Category.APIID)
	suite.Equal(query.Category.Name, models.Uncategorized)

	feeds := suite.db.FeedsFromCategory(query.Category.APIID, &suite.user)
	suite.NotEmpty(feeds)
	suite.Equal(feeds[0].Title, feed.Title)
	suite.Equal(feeds[0].ID, feed.ID)
	suite.Equal(feeds[0].APIID, feed.APIID)
}

func (suite *DatabaseTestSuite) TestNewFeedWithCategory() {
	ctg := suite.db.NewCategory("News", &suite.user)
	suite.NotEmpty(ctg.APIID)
	suite.NotZero(ctg.ID)
	suite.Empty(ctg.Feeds)

	feed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", ctg.APIID, &suite.user)
	suite.Require().Nil(err)

	query, found := suite.db.FeedWithAPIID(feed.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)

	suite.NotZero(query.Category.ID)
	suite.NotEmpty(query.Category.APIID)
	suite.Equal(query.Category.Name, "News")

	feeds := suite.db.FeedsFromCategory(ctg.APIID, &suite.user)
	suite.NotEmpty(feeds)
	suite.Equal(feeds[0].Title, feed.Title)
	suite.Equal(feeds[0].ID, feed.ID)
	suite.Equal(feeds[0].APIID, feed.APIID)
}

func (suite *DatabaseTestSuite) TestNewFeedWithNonExistingCategory() {
	_, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", createAPIID(), &suite.user)
	suite.Equal(ErrModelNotFound, err)
}

func (suite *DatabaseTestSuite) TestFeedsFromNonExistingCategory() {
	feeds := suite.db.FeedsFromCategory(createAPIID(), &suite.user)
	suite.Empty(feeds)
}

func (suite *DatabaseTestSuite) TestChangeFeedCategory() {
	firstCtg := suite.db.NewCategory("News", &suite.user)
	secondCtg := suite.db.NewCategory("Tech", &suite.user)

	feed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID, &suite.user)
	suite.Require().Nil(err)

	feeds := suite.db.FeedsFromCategory(firstCtg.APIID, &suite.user)
	suite.Require().Len(feeds, 1)
	suite.Equal(feeds[0].APIID, feed.APIID)
	suite.Equal(feeds[0].Title, feed.Title)

	feeds = suite.db.FeedsFromCategory(secondCtg.APIID, &suite.user)
	suite.Empty(feeds)

	err = suite.db.ChangeFeedCategory(feed.APIID, secondCtg.APIID, &suite.user)
	suite.Nil(err)

	feeds = suite.db.FeedsFromCategory(firstCtg.APIID, &suite.user)
	suite.Empty(feeds)

	feeds = suite.db.FeedsFromCategory(secondCtg.APIID, &suite.user)
	suite.Require().Len(feeds, 1)
	suite.Equal(feeds[0].APIID, feed.APIID)
	suite.Equal(feeds[0].Title, feed.Title)
}

func (suite *DatabaseTestSuite) TestChangeUnknownFeedCategory() {
	err := suite.db.ChangeFeedCategory("bogus", "none", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestChangeFeedCategoryToUnknown() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

	err := suite.db.ChangeFeedCategory(feed.APIID, "bogus", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestFeeds() {
	for i := 0; i < 5; i++ {
		feed := suite.db.NewFeed("Test site "+strconv.Itoa(i), "http://example.com", &suite.user)
		suite.Require().NotZero(feed.ID)
		suite.Require().NotEmpty(feed.APIID)
	}

	feeds := suite.db.Feeds(&suite.user)
	suite.Len(feeds, 5)
}

func (suite *DatabaseTestSuite) TestEditFeed() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

	query, found := suite.db.FeedWithAPIID(feed.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)

	feed.Title = "Testing New Name"
	feed.Subscription = "http://example.com/feed"

	err := suite.db.EditFeed(&feed, &suite.user)
	suite.Nil(err)

	query, found = suite.db.FeedWithAPIID(feed.APIID, &suite.user)
	suite.True(found)
	suite.Equal(feed.Title, "Testing New Name")
	suite.Equal(feed.Subscription, "http://example.com/feed")
}

func (suite *DatabaseTestSuite) TestEditNonExistingFeed() {
	err := suite.db.EditFeed(&models.Feed{}, &suite.user)
	suite.Equal(ErrModelNotFound, err)
}

func (suite *DatabaseTestSuite) TestDeleteFeed() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

	query, found := suite.db.FeedWithAPIID(feed.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.APIID)

	err := suite.db.DeleteFeed(feed.APIID, &suite.user)
	suite.Nil(err)

	_, found = suite.db.FeedWithAPIID(feed.APIID, &suite.user)
	suite.False(found)
}

func (suite *DatabaseTestSuite) TestDeleteNonExistingFeed() {
	err := suite.db.DeleteFeed(createAPIID(), &suite.user)
	suite.Equal(ErrModelNotFound, err)
}

func (suite *DatabaseTestSuite) TestNewTag() {
	tag := suite.db.NewTag("tech", &suite.user)

	query, found := suite.db.TagWithAPIID(tag.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.Name)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)
}

func (suite *DatabaseTestSuite) TestTagEntries() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

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

	entries, err := suite.db.NewEntries(entries, feed.APIID, &suite.user)
	suite.Require().NotEmpty(entries)
	suite.Require().Nil(err)

	entries = suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	tag := suite.db.NewTag("Tech", &suite.user)
	suite.Nil(err)
	suite.NotZero(tag.ID)

	dbTag, found := suite.db.TagWithAPIID(tag.APIID, &suite.user)
	suite.True(found)
	suite.Equal(tag.APIID, dbTag.APIID)
	suite.Equal(tag.Name, dbTag.Name)
	suite.NotZero(dbTag.ID)

	err = suite.db.TagEntries(tag.APIID, entryAPIIDs, &suite.user)
	suite.Nil(err)

	taggedEntries := suite.db.EntriesFromTag(tag.APIID, models.Any, true, &suite.user)
	suite.Nil(err)
	suite.Len(taggedEntries, 5)
}

func (suite *DatabaseTestSuite) TestTagMultipleEntries() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

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

	_, err := suite.db.NewEntries(entries, feed.APIID, &suite.user)
	suite.Require().Nil(err)

	secondTag := models.Tag{
		Name: "Second tag",
	}

	firstTag := suite.db.NewTag("First tag", &suite.user)
	suite.NotZero(firstTag.ID)

	secondTag = suite.db.NewTag("Second Tag", &suite.user)
	suite.NotZero(secondTag.ID)

	suite.Require().NotEqual(firstTag.APIID, secondTag.APIID)

	entries = suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = suite.db.TagEntries(firstTag.APIID, entryAPIIDs, &suite.user)
	suite.Nil(err)

	err = suite.db.TagEntries(secondTag.APIID, entryAPIIDs, &suite.user)
	suite.Nil(err)

	taggedEntries := suite.db.EntriesFromTag(firstTag.APIID, models.Any, true, &suite.user)
	suite.Len(taggedEntries, 5)

	taggedEntries = suite.db.EntriesFromTag(secondTag.APIID, models.Any, true, &suite.user)
	suite.Len(taggedEntries, 5)
}

func (suite *DatabaseTestSuite) TestEntriesFromMultipleTags() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

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

	entries, err := suite.db.NewEntries(entries, feed.APIID, &suite.user)
	suite.Require().Len(entries, 15)
	suite.Require().Nil(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		suite.Require().NotEmpty(entry.APIID)
		entryAPIIDs[i] = entry.APIID
	}

	firstTag := suite.db.NewTag("First tag", &suite.user)
	suite.NotZero(firstTag.ID)

	secondTag := suite.db.NewTag("Second tag", &suite.user)
	suite.NotZero(secondTag.ID)

	err = suite.db.TagEntries(firstTag.APIID, entryAPIIDs[0:5], &suite.user)
	suite.Nil(err)

	err = suite.db.TagEntries(secondTag.APIID, entryAPIIDs[5:10], &suite.user)
	suite.Nil(err)

	suite.Require().NotEqual(firstTag.APIID, secondTag.APIID)

	taggedEntries := suite.db.EntriesFromMultipleTags([]string{firstTag.APIID, secondTag.APIID}, true, models.Any, &suite.user)
	suite.Len(taggedEntries, 10)
}

func (suite *DatabaseTestSuite) TestDeleteTag() {
	tag := suite.db.NewTag("News", &suite.user)

	query, found := suite.db.TagWithAPIID(tag.APIID, &suite.user)
	suite.True(found)
	suite.NotEmpty(query.APIID)

	err := suite.db.DeleteTag(tag.APIID, &suite.user)
	suite.Nil(err)

	_, found = suite.db.TagWithAPIID(tag.APIID, &suite.user)
	suite.False(found)
}

func (suite *DatabaseTestSuite) TestEditTag() {
	tag := suite.db.NewTag("News", &suite.user)

	query, found := suite.db.TagWithAPIID(tag.APIID, &suite.user)
	suite.True(found)
	suite.Equal(query.Name, "News")

	err := suite.db.EditTagName(tag.APIID, "World News", &suite.user)
	suite.Require().Nil(err)

	query, found = suite.db.TagWithAPIID(tag.APIID, &suite.user)
	suite.True(found)
	suite.Equal(tag.ID, query.ID)
	suite.Equal(query.Name, "World News")
}

func (suite *DatabaseTestSuite) TestNewEntry() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.NotZero(feed.ID)
	suite.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(entry.APIID)

	query, found := suite.db.EntryWithAPIID(entry.APIID, &suite.user)
	suite.True(found)
	suite.NotZero(query.FeedID)

	entries := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.NotEmpty(entries)
	suite.Len(entries, 1)
	suite.Equal(entries[0].Title, entry.Title)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeedWithNonExistenFeed() {
	entries := suite.db.EntriesFromFeed(createAPIID(), true, models.Unread, &suite.user)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestNewEntryWithEmptyFeed() {
	entry := models.Entry{
		Title:     "Test Entry",
		Link:      "http://example.com",
		Author:    "varddum",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, createAPIID(), &suite.user)
	suite.NotNil(err)
	suite.Equal(ErrModelNotFound, err)
	suite.Zero(entry.ID)
	suite.Empty(entry.APIID)

	query, found := suite.db.EntryWithAPIID(entry.APIID, &suite.user)
	suite.False(found)
	suite.Zero(query.FeedID)
}

func (suite *DatabaseTestSuite) TestNewEntryWithBadFeed() {
	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, createAPIID(), &suite.user)
	suite.NotNil(err)
	suite.Zero(entry.ID)
	suite.Empty(entry.APIID)

	query, found := suite.db.EntryWithAPIID(entry.APIID, &suite.user)
	suite.False(found)
	suite.Zero(query.FeedID)
}

func (suite *DatabaseTestSuite) TestNewEntries() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.NotZero(feed.ID)
	suite.NotEmpty(feed.APIID)

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

	entries, err := suite.db.NewEntries(entries, feed.APIID, &suite.user)
	suite.Require().Len(entries, 5)
	suite.Require().Nil(err)

	entries = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Len(entries, 5)
	for _, entry := range entries {
		suite.NotZero(entry.ID)
		suite.NotZero(entry.Title)
	}
}

func (suite *DatabaseTestSuite) TestNewEntriesWithBadFeed() {
	entries := []models.Entry{
		{},
	}
	_, err := suite.db.NewEntries(entries, "", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestEntries() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
	suite.Require().Nil(err)

	entries := suite.db.Entries(true, models.Unread, &suite.user)
	suite.NotEmpty(entries)
	suite.Equal(entries[0].Title, entry.Title)
}

func (suite *DatabaseTestSuite) TestEntriesWithNoneMarker() {
	entries := suite.db.Entries(true, models.None, &suite.user)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeed() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
	suite.Require().Nil(err)

	entries := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().NotEmpty(entries)
	suite.Equal(entries[0].Title, entry.Title)

	entries = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeedWithNoneMarker() {
	entries := suite.db.EntriesFromFeed("bogus", true, models.None, &suite.user)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestEntryWithGUIDExists() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

	entry := models.Entry{
		Title:     "Test Entry",
		GUID:      "entry@test",
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
	suite.Nil(err)
	suite.True(suite.db.EntryWithGUIDExists(entry.GUID, feed.APIID, &suite.user))
}

func (suite *DatabaseTestSuite) TestEntryWithGUIDDoesNotExists() {
	feed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)

	entry := models.Entry{
		Title:     "Test Entry",
		Published: time.Now(),
	}

	_, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
	suite.Nil(err)
	suite.False(suite.db.EntryWithGUIDExists("item@test", feed.APIID, &suite.user))
}

func (suite *DatabaseTestSuite) TestEntriesFromCategory() {
	firstCtg := suite.db.NewCategory("News", &suite.user)
	suite.NotEmpty(firstCtg.APIID)

	secondCtg := suite.db.NewCategory("Tech", &suite.user)
	suite.NotEmpty(secondCtg.APIID)

	firstFeed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID, &suite.user)
	suite.Nil(err)

	secondFeed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID, &suite.user)
	suite.Nil(err)

	thirdFeed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID, &suite.user)
	suite.Nil(err)

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

			_, err = suite.db.NewEntry(entry, firstFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		} else if i < 7 {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err = suite.db.NewEntry(entry, secondFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Third Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Unread,
				Published: time.Now(),
			}

			_, err = suite.db.NewEntry(entry, thirdFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		}

	}

	entries := suite.db.EntriesFromCategory(firstCtg.APIID, false, models.Unread, &suite.user)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	suite.Equal(entries[0].Title, "First Feed Test Entry 0")

	entries = suite.db.EntriesFromCategory(secondCtg.APIID, true, models.Unread, &suite.user)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	suite.Equal(entries[0].Title, "Third Feed Test Entry 9")
	suite.Equal(entries[len(entries)-1].Title, "Second Feed Test Entry 5")
}

func (suite *DatabaseTestSuite) TestEntriesFromCategoryWithtNoneMarker() {
	entries := suite.db.EntriesFromCategory("bogus", true, models.None, &suite.user)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestEntriesFromNonExistingCategory() {
	entries := suite.db.EntriesFromCategory(createAPIID(), true, models.Unread, &suite.user)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestMarkCategory() {
	firstCtg := suite.db.NewCategory("News", &suite.user)
	suite.NotEmpty(firstCtg.APIID)

	secondCtg := suite.db.NewCategory("Tech", &suite.user)
	suite.NotEmpty(secondCtg.APIID)

	firstFeed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstFeed.APIID)

	secondFeed, err := suite.db.NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstFeed.APIID)

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

			_, err = suite.db.NewEntry(entry, firstFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Read,
				Published: time.Now(),
			}

			_, err = suite.db.NewEntry(entry, secondFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		}

	}

	suite.Require().Equal(suite.db.db.Model(&suite.user).Association("Entries").Count(), 10)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err = suite.db.MarkCategory(firstCtg.APIID, models.Read, &suite.user)
	suite.Nil(err)

	entries := suite.db.EntriesFromCategory(firstCtg.APIID, true, models.Any, &suite.user)
	suite.Len(entries, 5)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Read)
	}

	err = suite.db.MarkCategory(secondCtg.APIID, models.Unread, &suite.user)
	suite.Nil(err)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries = suite.db.EntriesFromCategory(secondCtg.APIID, true, models.Any, &suite.user)
	suite.Len(entries, 5)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Unread)
	}
}

func (suite *DatabaseTestSuite) TestMarkFeed() {
	firstFeed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.NotEmpty(firstFeed.APIID)

	secondFeed := suite.db.NewFeed("Test site", "http://example.com", &suite.user)
	suite.NotEmpty(secondFeed.APIID)

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

			_, err := suite.db.NewEntry(entry, firstFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "varddum",
				Link:      "http://example.com",
				Mark:      models.Read,
				Published: time.Now(),
			}

			_, err := suite.db.NewEntry(entry, secondFeed.APIID, &suite.user)
			suite.Require().Nil(err)
		}

	}

	suite.Require().Equal(suite.db.db.Model(&suite.user).Association("Entries").Count(), 10)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err := suite.db.MarkFeed(firstFeed.APIID, models.Read, &suite.user)
	suite.Nil(err)

	entries := suite.db.EntriesFromFeed(firstFeed.APIID, true, models.Read, &suite.user)
	suite.Len(entries, 5)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Read)
	}

	err = suite.db.MarkFeed(secondFeed.APIID, models.Unread, &suite.user)
	suite.Nil(err)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries = suite.db.EntriesFromFeed(secondFeed.APIID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Unread)
	}
}

func (suite *DatabaseTestSuite) TestMarkEntry() {
	feed := suite.db.NewFeed("News", "http://localhost/news", &suite.user)

	entry := models.Entry{
		Title:     "Article",
		Mark:      models.Unread,
		Published: time.Now(),
	}

	entry, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
	suite.Require().Nil(err)

	entries := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Len(entries, 1)

	err = suite.db.MarkEntry(entry.APIID, models.Read, &suite.user)
	suite.Require().Nil(err)

	entries = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Len(entries, 0)

	entries = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Require().Len(entries, 1)
}

func (suite *DatabaseTestSuite) TestStats() {
	feed := suite.db.NewFeed("News", "http://example.com", &suite.user)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		_, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
		suite.Require().Nil(err)
	}

	stats := suite.db.Stats(&suite.user)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestFeedStats() {
	feed := suite.db.NewFeed("News", "http://example.com", &suite.user)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		_, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		_, err := suite.db.NewEntry(entry, feed.APIID, &suite.user)
		suite.Require().Nil(err)
	}

	stats := suite.db.FeedStats(feed.APIID, &suite.user)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestCategoryStats() {
	ctg := suite.db.NewCategory("World", &suite.user)
	suite.Require().NotEmpty(ctg.APIID)

	feed, err := suite.db.NewFeedWithCategory("News", "http://example.com", ctg.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		_, err = suite.db.NewEntry(entry, feed.APIID, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.Unread,
			Published: time.Now(),
		}

		_, err = suite.db.NewEntry(entry, feed.APIID, &suite.user)
		suite.Require().Nil(err)
	}

	stats := suite.db.CategoryStats(ctg.APIID, &suite.user)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestKeyBelongsToUser() {
	key, err := suite.db.NewAPIKey("secret", &suite.user)
	suite.Require().Nil(err)

	found := suite.db.KeyBelongsToUser(models.APIKey{Key: key.Key}, &suite.user)
	suite.True(found)
}

func (suite *DatabaseTestSuite) TestKeyDoesNotBelongToUser() {
	key := models.APIKey{
		Key: "123456789",
	}

	found := suite.db.KeyBelongsToUser(key, &suite.user)
	suite.False(found)
}

func TestNewDB(t *testing.T) {
	_, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	assert.Nil(t, err)
	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestNewDBWithBadOptions(t *testing.T) {
	_, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "bogus",
	})
	assert.NotNil(t, err)
}

func TestNewUser(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUsers(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test_one", "golang")
	db.NewUser("test_two", "password")

	users := db.Users()
	assert.Len(t, users, 2)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUsersWithFields(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test_one", "golang")
	db.NewUser("test_two", "password")

	users := db.Users("uncategorized_category_api_id")
	assert.Len(t, users, 2)
	assert.NotEmpty(t, users[0].UncategorizedCategoryAPIID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDeleteUser(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("first", "golang")
	db.NewUser("second", "password")

	users := db.Users()
	assert.Len(t, users, 2)

	err = db.DeleteUser(users[0].APIID)
	assert.Nil(t, err)

	users = db.Users()
	assert.Len(t, users, 1)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDeleteUnknownUser(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	err = db.DeleteUser("bogus")
	assert.Equal(t, ErrModelNotFound, err)
}

func TestChangeUserName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")
	user, found := db.UserWithName("test")
	require.True(t, found)

	err = db.ChangeUserName(user.APIID, "new_name")
	require.Nil(t, err)

	user, found = db.UserWithName("test")
	assert.False(t, found)
	assert.Zero(t, user.ID)

	user, found = db.UserWithName("new_name")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestChangeUnknownUserName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	err = db.ChangeUserName("bogus", "none")
	assert.Equal(t, ErrModelNotFound, err)
}

func TestChangeUserPassword(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")
	user, found := db.UserWithCredentials("test", "golang")
	require.True(t, found)

	db.ChangeUserPassword(user.APIID, "new_password")

	_, found = db.UserWithCredentials("test", "golang")
	assert.False(t, found)

	user, found = db.UserWithCredentials("test", "new_password")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestSuccessfulAuthentication(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	user := db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	user, found = db.UserWithCredentials("test", "golang")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestBadPasswordAuthentication(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	user, found = db.UserWithCredentials("test", "badpass")
	assert.False(t, found)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithAPIID(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	require.True(t, found)

	userWithID, found := db.UserWithAPIID(user.APIID)
	assert.True(t, found)
	assert.Equal(t, user.APIID, userWithID.APIID)
	assert.Equal(t, user.ID, userWithID.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithUnknownAPIID(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	userWithID, found := db.UserWithAPIID("bogus")
	assert.False(t, found)
	assert.Zero(t, userWithID.APIID)
	assert.Zero(t, userWithID.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.NotZero(t, user.ID)
	assert.NotZero(t, user.APIID)
	assert.Equal(t, user.Username, "test")

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithUnknownName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("bogus")
	assert.False(t, found)
	assert.Zero(t, user.ID)
	assert.Zero(t, user.APIID)
	assert.NotEqual(t, user.Username, "test")

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
