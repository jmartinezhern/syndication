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
	suite.db, err = NewDB("sqlite3", TestDatabasePath)
	suite.Require().NotNil(suite.db)
	suite.Require().Nil(err)

	err = suite.db.NewUser("test", "golang")
	suite.Require().Nil(err)

	suite.user, err = suite.db.UserWithName("test")
	suite.Require().Nil(err)
}

func (suite *DatabaseTestSuite) TearDownTest() {
	err := suite.db.Close()
	suite.Nil(err)
	err = os.Remove(suite.db.Connection)
	suite.Nil(err)
}

func (suite *DatabaseTestSuite) TestNewCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(ctg.APIID)
	suite.NotZero(ctg.ID)
	suite.NotZero(ctg.UserID)
	suite.NotZero(ctg.CreatedAt)
	suite.NotZero(ctg.UpdatedAt)
	suite.NotZero(ctg.UserID)

	query, err := suite.db.Category(ctg.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Name)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)
}

func (suite *DatabaseTestSuite) TestNewCategoryWithNoName() {
	ctg := models.Category{
		Name: "",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestCategories() {
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			Name: "Test Category " + strconv.Itoa(i),
		}

		err := suite.db.NewCategory(&ctg, &suite.user)
		suite.Require().Nil(err)
	}

	ctgs := suite.db.Categories(&suite.user)
	suite.Len(ctgs, 7)
}

func (suite *DatabaseTestSuite) TestEditCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Category(ctg.APIID, &suite.user)
	suite.Nil(err)
	suite.Equal(query.Name, "News")

	ctg.Name = "World News"
	err = suite.db.EditCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	query, err = suite.db.Category(ctg.APIID, &suite.user)
	suite.Nil(err)
	suite.Equal(ctg.ID, query.ID)
	suite.Equal(query.Name, "World News")
}

func (suite *DatabaseTestSuite) TestEditNonExistingCategory() {
	err := suite.db.EditCategory(&models.Category{}, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Category(ctg.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.APIID)

	err = suite.db.DeleteCategory(ctg.APIID, &suite.user)
	suite.Nil(err)

	_, err = suite.db.Category(ctg.APIID, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteNonExistingCategory() {
	err := suite.db.DeleteCategory(createAPIID(), &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteSystemCategory() {
	err := suite.db.DeleteCategory(suite.user.SavedCategoryAPIID, &suite.user)
	suite.IsType(BadRequest{}, err)
}

func (suite *DatabaseTestSuite) TestNewFeedWithDefaults() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)

	suite.NotZero(query.Category.ID)
	suite.NotEmpty(query.Category.APIID)
	suite.Equal(query.Category.Name, models.Uncategorized)

	feeds, err := suite.db.FeedsFromCategory(query.Category.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(feeds)
	suite.Equal(feeds[0].Title, feed.Title)
	suite.Equal(feeds[0].ID, feed.ID)
	suite.Equal(feeds[0].APIID, feed.APIID)
}

func (suite *DatabaseTestSuite) TestNewFeedWithCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(ctg.APIID)
	suite.NotZero(ctg.ID)
	suite.Empty(ctg.Feeds)

	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     ctg,
		CategoryID:   ctg.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)

	suite.NotZero(query.Category.ID)
	suite.NotEmpty(query.Category.APIID)
	suite.Equal(query.Category.Name, "News")

	feeds, err := suite.db.FeedsFromCategory(ctg.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(feeds)
	suite.Equal(feeds[0].Title, feed.Title)
	suite.Equal(feeds[0].ID, feed.ID)
	suite.Equal(feeds[0].APIID, feed.APIID)
}

func (suite *DatabaseTestSuite) TestNewFeedWithNonExistingCategory() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category: models.Category{
			APIID: createAPIID(),
		},
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.IsType(BadRequest{}, err)
}

func (suite *DatabaseTestSuite) TestFeedsFromNonExistingCategory() {
	_, err := suite.db.FeedsFromCategory(createAPIID(), &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestChangeFeedCategory() {
	firstCtg := models.Category{
		Name: "News",
	}

	secondCtg := models.Category{
		Name: "Tech",
	}

	err := suite.db.NewCategory(&firstCtg, &suite.user)
	err = suite.db.NewCategory(&secondCtg, &suite.user)

	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     firstCtg,
		CategoryID:   firstCtg.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	feeds, err := suite.db.FeedsFromCategory(firstCtg.APIID, &suite.user)
	suite.Nil(err)
	suite.Require().Len(feeds, 1)
	suite.Equal(feeds[0].APIID, feed.APIID)
	suite.Equal(feeds[0].Title, feed.Title)

	feeds, err = suite.db.FeedsFromCategory(secondCtg.APIID, &suite.user)
	suite.Nil(err)
	suite.Empty(feeds)

	err = suite.db.ChangeFeedCategory(feed.APIID, secondCtg.APIID, &suite.user)
	suite.Nil(err)

	feeds, err = suite.db.FeedsFromCategory(firstCtg.APIID, &suite.user)
	suite.Nil(err)
	suite.Empty(feeds)

	feeds, err = suite.db.FeedsFromCategory(secondCtg.APIID, &suite.user)
	suite.Nil(err)
	suite.Require().Len(feeds, 1)
	suite.Equal(feeds[0].APIID, feed.APIID)
	suite.Equal(feeds[0].Title, feed.Title)
}

func (suite *DatabaseTestSuite) TestChangeUnknownFeedCategory() {
	err := suite.db.ChangeFeedCategory("bogus", "none", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestChangeFeedCategoryToUnknown() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	err = suite.db.ChangeFeedCategory(feed.APIID, "bogus", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestFeeds() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			Title:        "Test site " + strconv.Itoa(i),
			Subscription: "http://example.com",
		}

		err := suite.db.NewFeed(&feed, &suite.user)
		suite.Require().Nil(err)
	}

	feeds := suite.db.Feeds(&suite.user)
	suite.Len(feeds, 5)
}

func (suite *DatabaseTestSuite) TestEditFeed() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)

	feed.Title = "Testing New Name"
	feed.Subscription = "http://example.com/feed"

	err = suite.db.EditFeed(&feed, &suite.user)
	suite.Nil(err)

	query, err = suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)
	suite.Equal(feed.Title, "Testing New Name")
	suite.Equal(feed.Subscription, "http://example.com/feed")
}

func (suite *DatabaseTestSuite) TestEditNonExistingFeed() {
	err := suite.db.EditFeed(&models.Feed{}, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestConflictingNewCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	err = suite.db.NewCategory(&ctg, &suite.user)
	suite.IsType(Conflict{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteFeed() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.APIID)

	err = suite.db.DeleteFeed(feed.APIID, &suite.user)
	suite.Nil(err)

	_, err = suite.db.Feed(feed.APIID, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteNonExistingFeed() {
	err := suite.db.DeleteFeed(createAPIID(), &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestNewEntry() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotZero(feed.ID)
	suite.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "varddum",
		Link:        "http://example.com",
		Mark:        models.Unread,
		Feed:        feed,
		Published:   time.Now(),
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)
	suite.NotZero(entry.ID)
	suite.NotEmpty(entry.APIID)

	query, err := suite.db.Entry(entry.APIID, &suite.user)
	suite.Nil(err)
	suite.NotZero(query.FeedID)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 1)
	suite.Equal(entries[0].ID, entry.ID)
	suite.Equal(entries[0].Title, entry.Title)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeedWithNonExistenFeed() {
	_, err := suite.db.EntriesFromFeed(createAPIID(), true, models.Unread, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestNewEntryWithEmptyFeed() {
	entry := models.Entry{
		Title:       "Test Entry",
		Link:        "http://example.com",
		Description: "Testing entry",
		Author:      "varddum",
		Mark:        models.Unread,
		Published:   time.Now(),
	}

	err := suite.db.NewEntry(&entry, &suite.user)
	suite.IsType(BadRequest{}, err)
	suite.Zero(entry.ID)
	suite.Empty(entry.APIID)

	query, err := suite.db.Entry(entry.APIID, &suite.user)
	suite.NotNil(err)
	suite.Zero(query.FeedID)
}

func (suite *DatabaseTestSuite) TestNewEntryWithBadFeed() {
	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "varddum",
		Mark:        models.Unread,
		Published:   time.Now(),
		Feed: models.Feed{
			APIID: createAPIID(),
		},
	}

	err := suite.db.NewEntry(&entry, &suite.user)
	suite.NotNil(err)
	suite.Zero(entry.ID)
	suite.Empty(entry.APIID)

	query, err := suite.db.Entry(entry.APIID, &suite.user)
	suite.NotNil(err)
	suite.Zero(query.FeedID)
}

func (suite *DatabaseTestSuite) TestNewEntries() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotZero(feed.ID)
	suite.NotEmpty(feed.APIID)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:       "Test Entry",
			Description: "Testing entry",
			Author:      "varddum",
			Link:        "http://example.com",
			Mark:        models.Unread,
			Feed:        feed,
			Published:   time.Now(),
		}

		entries = append(entries, entry)
	}

	err = suite.db.NewEntries(entries, &feed, &suite.user)
	suite.Require().Nil(err)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	for _, entry := range entries {
		suite.NotZero(entry.ID)
		suite.NotZero(entry.Title)
	}
}

func (suite *DatabaseTestSuite) TestNewEntriesWithNoFeed() {
	err := suite.db.NewEntries(nil, &models.Feed{}, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestNewEntriesWithUnknownFeed() {
	err := suite.db.NewEntries([]models.Entry{{}}, &models.Feed{APIID: "bogus"}, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestNewEntriesWithEmptyArray() {
	err := suite.db.NewEntries([]models.Entry{}, &models.Feed{APIID: "bogus"}, &suite.user)
	suite.Nil(err)
}

func (suite *DatabaseTestSuite) TestEntries() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "varddum",
		Link:        "http://example.com",
		Mark:        models.Unread,
		Feed:        feed,
		Published:   time.Now(),
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.Entries(true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Equal(entries[0].ID, entry.ID)
	suite.Equal(entries[0].Title, entry.Title)
}

func (suite *DatabaseTestSuite) TestEntriesWithNoneMarker() {
	_, err := suite.db.Entries(true, models.None, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeed() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "varddum",
		Link:        "http://example.com",
		Mark:        models.Unread,
		Feed:        feed,
		Published:   time.Now(),
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.Require().NotEmpty(entries)
	suite.Equal(entries[0].ID, entry.ID)
	suite.Equal(entries[0].Title, entry.Title)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Nil(err)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeedWithNoneMarker() {
	_, err := suite.db.EntriesFromFeed("bogus", true, models.None, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestEntryWithGUIDExists() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entry := models.Entry{
		Title:     "Test Entry",
		GUID:      "entry@test",
		Feed:      feed,
		Published: time.Now(),
	}

	err = suite.db.NewEntry(&entry, &suite.user)

	suite.True(suite.db.EntryWithGUIDExists(entry.GUID, feed.APIID, &suite.user))
}

func (suite *DatabaseTestSuite) TestEntryWithGUIDDoesNotExists() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entry := models.Entry{
		Title:     "Test Entry",
		Feed:      feed,
		Published: time.Now(),
	}

	err = suite.db.NewEntry(&entry, &suite.user)

	suite.False(suite.db.EntryWithGUIDExists("item@test", feed.APIID, &suite.user))
}

func (suite *DatabaseTestSuite) TestEntriesFromCategory() {
	firstCtg := models.Category{
		Name: "News",
	}

	secondCtg := models.Category{
		Name: "Tech",
	}

	err := suite.db.NewCategory(&firstCtg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstCtg.APIID)

	err = suite.db.NewCategory(&secondCtg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(secondCtg.APIID)

	firstFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     firstCtg,
		CategoryID:   firstCtg.ID,
	}

	secondFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     secondCtg,
		CategoryID:   secondCtg.ID,
	}

	thirdFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     secondCtg,
		CategoryID:   secondCtg.ID,
	}

	err = suite.db.NewFeed(&firstFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstFeed.APIID)

	err = suite.db.NewFeed(&secondFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(secondFeed.APIID)

	err = suite.db.NewFeed(&thirdFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(secondFeed.APIID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:       "First Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "varddum",
				Link:        "http://example.com",
				Mark:        models.Unread,
				Feed:        firstFeed,
				Published:   time.Now(),
			}
		} else {
			if i < 7 {
				entry = models.Entry{
					Title:       "Second Feed Test Entry " + strconv.Itoa(i),
					Description: "Testing entry " + strconv.Itoa(i),
					Author:      "varddum",
					Link:        "http://example.com",
					Mark:        models.Unread,
					Feed:        secondFeed,
					Published:   time.Now(),
				}
			} else {
				entry = models.Entry{
					Title:       "Third Feed Test Entry " + strconv.Itoa(i),
					Description: "Testing entry " + strconv.Itoa(i),
					Author:      "varddum",
					Link:        "http://example.com",
					Mark:        models.Unread,
					Feed:        thirdFeed,
					Published:   time.Now(),
				}
			}
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	entries, err := suite.db.EntriesFromCategory(firstCtg.APIID, false, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	suite.Equal(entries[0].Title, "First Feed Test Entry 0")

	entries, err = suite.db.EntriesFromCategory(secondCtg.APIID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	suite.Equal(entries[0].Title, "Third Feed Test Entry 9")
	suite.Equal(entries[len(entries)-1].Title, "Second Feed Test Entry 5")
}

func (suite *DatabaseTestSuite) TestEntriesFromCategoryWithtNoneMarker() {
	_, err := suite.db.EntriesFromCategory("bogus", true, models.None, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestEntriesFromNonExistingCategory() {
	_, err := suite.db.EntriesFromCategory(createAPIID(), true, models.Unread, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestMarkCategory() {
	firstCtg := models.Category{
		Name: "News",
	}

	secondCtg := models.Category{
		Name: "Tech",
	}

	err := suite.db.NewCategory(&firstCtg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstCtg.APIID)

	err = suite.db.NewCategory(&secondCtg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(secondCtg.APIID)

	firstFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     firstCtg,
		CategoryID:   firstCtg.ID,
	}

	secondFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     secondCtg,
		CategoryID:   secondCtg.ID,
	}

	err = suite.db.NewFeed(&firstFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstFeed.APIID)

	err = suite.db.NewFeed(&secondFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(secondFeed.APIID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:       "First Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "varddum",
				Link:        "http://example.com",
				Mark:        models.Unread,
				Feed:        firstFeed,
				Published:   time.Now(),
			}
		} else {
			entry = models.Entry{
				Title:       "Second Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "varddum",
				Link:        "http://example.com",
				Mark:        models.Read,
				Feed:        secondFeed,
				Published:   time.Now(),
			}
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	suite.Require().Equal(suite.db.db.Model(&suite.user).Association("Entries").Count(), 10)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err = suite.db.MarkCategory(firstCtg.APIID, models.Read, &suite.user)
	suite.Nil(err)

	entries, err := suite.db.EntriesFromCategory(firstCtg.APIID, true, models.Any, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Read)
	}

	err = suite.db.MarkCategory(secondCtg.APIID, models.Unread, &suite.user)
	suite.Nil(err)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries, err = suite.db.EntriesFromCategory(secondCtg.APIID, true, models.Any, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Unread)
	}
}

func (suite *DatabaseTestSuite) TestMarkFeed() {
	firstFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	secondFeed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&firstFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(firstFeed.APIID)

	err = suite.db.NewFeed(&secondFeed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(secondFeed.APIID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:       "First Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "varddum",
				Link:        "http://example.com",
				Mark:        models.Unread,
				Feed:        firstFeed,
				Published:   time.Now(),
			}
		} else {
			entry = models.Entry{
				Title:       "Second Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "varddum",
				Link:        "http://example.com",
				Mark:        models.Read,
				Feed:        secondFeed,
				Published:   time.Now(),
			}
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	suite.Require().Equal(suite.db.db.Model(&suite.user).Association("Entries").Count(), 10)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err = suite.db.MarkFeed(firstFeed.APIID, models.Read, &suite.user)
	suite.Nil(err)

	entries, err := suite.db.EntriesFromFeed(firstFeed.APIID, true, models.Read, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Read)
	}

	err = suite.db.MarkFeed(secondFeed.APIID, models.Unread, &suite.user)
	suite.Nil(err)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries, err = suite.db.EntriesFromFeed(secondFeed.APIID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Unread)
	}
}

func (suite *DatabaseTestSuite) TestMarkEntry() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://localhost/news",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entry := models.Entry{
		Title:     "Article",
		Feed:      feed,
		Mark:      models.Unread,
		Published: time.Now(),
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)

	err = suite.db.MarkEntry(entry.APIID, models.Read, &suite.user)
	suite.Require().Nil(err)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)
}

func (suite *DatabaseTestSuite) TestStats() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Feed:      feed,
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Feed:      feed,
			Mark:      models.Unread,
			Published: time.Now(),
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	stats := suite.db.Stats(&suite.user)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestFeedStats() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Feed:      feed,
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Feed:      feed,
			Mark:      models.Unread,
			Published: time.Now(),
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	stats, err := suite.db.FeedStats(feed.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestFeedStatsForUnknownFeed() {
	_, err := suite.db.FeedStats("bogus", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestCategoryStats() {
	category := models.Category{
		Name: "World",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.APIID)

	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Feed:      feed,
			Mark:      models.Read,
			Saved:     true,
			Published: time.Now(),
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Feed:      feed,
			Mark:      models.Unread,
			Published: time.Now(),
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	stats, err := suite.db.CategoryStats(category.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestCategoryStatsForNonExistentCategory() {
	_, err := suite.db.CategoryStats("bogus", &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestKeyBelongsToUser() {
	key, err := suite.db.NewAPIKey("secret", &suite.user)
	suite.Require().Nil(err)

	found, err := suite.db.KeyBelongsToUser(&models.APIKey{Key: key.Key}, &suite.user)
	suite.Nil(err)
	suite.True(found)
}

func (suite *DatabaseTestSuite) TestNoKeyBelongsToUser() {
	_, err := suite.db.KeyBelongsToUser(&models.APIKey{Key: ""}, &suite.user)
	suite.NotNil(err)
}

func (suite *DatabaseTestSuite) TestKeyDoesNotBelongToUser() {
	key := models.APIKey{
		Key: "123456789",
	}

	found, err := suite.db.KeyBelongsToUser(&key, &suite.user)
	suite.Require().Nil(err)
	suite.False(found)
}

func (suite *DatabaseTestSuite) TestErrors() {
	conflictErr := Conflict{"Conflict Error"}
	suite.Equal(conflictErr.Code(), 409)
	suite.Equal(conflictErr.Error(), "Conflict Error")
	suite.Equal(conflictErr.String(), "Conflict")

	notFoundErr := NotFound{"NotFound Error"}
	suite.Equal(notFoundErr.Code(), 404)
	suite.Equal(notFoundErr.Error(), "NotFound Error")
	suite.Equal(notFoundErr.String(), "Not Found")

	badRequestErr := BadRequest{"BadRequest Error"}
	suite.Equal(badRequestErr.Code(), 400)
	suite.Equal(badRequestErr.Error(), "BadRequest Error")
	suite.Equal(badRequestErr.String(), "Bad Request")

	unauthorizedErr := Unauthorized{"Unauthorized Error"}
	suite.Equal(unauthorizedErr.Code(), 401)
	suite.Equal(unauthorizedErr.Error(), "Unauthorized Error")
	suite.Equal(unauthorizedErr.String(), "Unauthorized")
}

func TestNewDB(t *testing.T) {
	_, err := NewDB("sqlite3", TestDatabasePath)
	assert.Nil(t, err)
	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestNewDBWithBadOptions(t *testing.T) {
	_, err := NewDB("bogus", TestDatabasePath)
	assert.NotNil(t, err)
}

func TestNewUser(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	user, err := db.UserWithName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUsers(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test_one", "golang")
	assert.Nil(t, err)

	err = db.NewUser("test_two", "password")
	assert.Nil(t, err)

	users := db.Users()
	assert.Len(t, users, 2)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUsersWithFields(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test_one", "golang")
	assert.Nil(t, err)

	err = db.NewUser("test_two", "password")
	assert.Nil(t, err)

	users := db.Users("uncategorized_category_api_id", "saved_category_api_id")
	assert.Len(t, users, 2)
	assert.NotEmpty(t, users[0].SavedCategoryAPIID)
	assert.NotEmpty(t, users[0].UncategorizedCategoryAPIID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestNewConflictingUsers(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	err = db.NewUser("test", "password")
	assert.IsType(t, Conflict{}, err)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDeleteUser(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("first", "golang")
	assert.Nil(t, err)

	err = db.NewUser("second", "password")
	assert.Nil(t, err)

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
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.DeleteUser("bogus")
	assert.NotNil(t, err)
}

func TestChangeUserName(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	user, err := db.UserWithName("test")
	require.Nil(t, err)

	err = db.ChangeUserName(user.APIID, "new_name")
	require.Nil(t, err)

	user, err = db.UserWithName("test")
	assert.IsType(t, err, NotFound{})
	assert.Zero(t, user.ID)

	user, err = db.UserWithName("new_name")
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestChangeUnknownUserName(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.ChangeUserName("bogus", "none")
	assert.NotNil(t, err)
}

func TestChangeUserPassword(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	user, err := db.Authenticate("test", "golang")
	require.Nil(t, err)

	err = db.ChangeUserPassword(user.APIID, "new_password")
	assert.Nil(t, err)

	_, err = db.Authenticate("test", "golang")
	assert.IsType(t, err, Unauthorized{})

	user, err = db.Authenticate("test", "new_password")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestChangeUnknownUserPassword(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.ChangeUserPassword("bogus", "none")
	assert.NotNil(t, err)
}

func TestSuccessfulAuthentication(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	user, err := db.UserWithName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	user, err = db.Authenticate("test", "golang")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestBadPasswordAuthentication(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	user, err := db.UserWithName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	user, err = db.Authenticate("test", "badpass")
	assert.IsType(t, Unauthorized{}, err)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestBadUserAuthentication(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	_, err = db.Authenticate("test", "golang")
	assert.IsType(t, Unauthorized{}, err)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithAPIID(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	user, err := db.UserWithName("test")
	require.Nil(t, err)

	userWithID, err := db.UserWithAPIID(user.APIID)
	assert.Nil(t, err)
	assert.Equal(t, user.APIID, userWithID.APIID)
	assert.Equal(t, user.ID, userWithID.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithUnknownAPIID(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	userWithID, err := db.UserWithAPIID("bogus")
	assert.IsType(t, err, NotFound{})
	assert.Zero(t, userWithID.APIID)
	assert.Zero(t, userWithID.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithName(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	user, err := db.UserWithName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)
	assert.NotZero(t, user.APIID)
	assert.Equal(t, user.Username, "test")

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithUnknownName(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	user, err := db.UserWithName("bogus")
	assert.IsType(t, err, NotFound{})
	assert.Zero(t, user.ID)
	assert.Zero(t, user.APIID)
	assert.NotEqual(t, user.Username, "test")

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithPrimaryKey(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	user, err := db.UserWithName("test")
	require.Nil(t, err)

	userID, err := db.UserPrimaryKey(user.APIID)
	assert.Equal(t, user.ID, userID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUnknownUserWithPrimaryKey(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)

	err = db.NewUser("test", "golang")
	require.Nil(t, err)

	userID, err := db.UserPrimaryKey("bogus")
	assert.IsType(t, err, NotFound{})
	assert.Zero(t, userID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
