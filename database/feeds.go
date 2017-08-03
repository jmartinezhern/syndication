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
	"github.com/jmartinezhern/syndication/models"
)

// CreateFeed creates a new feed associated to a category with the given API ID
func (db *DB) CreateFeed(feed *models.Feed, ctgID string, user models.User) error {
	ctg, found := db.CategoryWithAPIID(ctgID, user)
	if !found {
		return ErrModelNotFound
	}

	feed.Category = ctg
	feed.CategoryID = ctg.ID
	feed.Category.APIID = ctg.APIID

	db.db.Model(&user).Association("Feeds").Append(feed)
	db.db.Model(&ctg).Association("Feeds").Append(feed)

	return nil
}

// CreateFeed creates a new feed associated to a category with the given API ID
func CreateFeed(feed *models.Feed, ctgID string, user models.User) error {
	return defaultInstance.CreateFeed(feed, ctgID, user)
}

// Feeds returns a list of all Feeds owned by a user
func (db *DB) Feeds(continuationID string, count int, user models.User) (feeds []models.Feed, next string) {
	query := db.db.Model(&user)

	if continuationID != "" {
		feed, found := FeedWithAPIID(continuationID, user)
		if found {
			query = query.Where("id >= ?", feed.ID)
		}
	}

	query.Limit(count + 1).Association("Feeds").Find(&feeds)

	if len(feeds) > count {
		next = feeds[len(feeds)-1].APIID
		feeds = feeds[:len(feeds)-1]
	}

	return
}

// Feeds returns a list of all Feeds owned by a user
func Feeds(continuationID string, count int, user models.User) ([]models.Feed, string) {
	return defaultInstance.Feeds(continuationID, count, user)
}

// FeedWithAPIID returns a Feed with id and owned by user
func (db *DB) FeedWithAPIID(id string, user models.User) (models.Feed, bool) {
	feed := models.Feed{}

	if !db.db.Model(&user).Where("api_id = ?", id).Related(&feed).RecordNotFound() {
		db.db.Model(&feed).Related(&feed.Category)
		return feed, true
	}

	return models.Feed{}, false
}

// FeedWithAPIID returns a Feed with id and owned by user
func FeedWithAPIID(id string, user models.User) (models.Feed, bool) {
	return defaultInstance.FeedWithAPIID(id, user)
}

// DeleteFeed with id and owned by user
func (db *DB) DeleteFeed(id string, user models.User) error {
	foundFeed := &models.Feed{}
	if !db.db.Model(&user).Where("api_id = ?", id).Related(foundFeed).RecordNotFound() {
		db.db.Delete(foundFeed)
		return nil
	}
	return ErrModelNotFound
}

// DeleteFeed with id and owned by user
func DeleteFeed(id string, user models.User) error {
	return defaultInstance.DeleteFeed(id, user)
}

// EditFeed owned by user
func (db *DB) EditFeed(id string, newFeed models.Feed, user models.User) (models.Feed, error) {
	if feed, found := db.FeedWithAPIID(id, user); found {
		db.db.Model(&feed).Updates(newFeed)
		return feed, nil
	}
	return models.Feed{}, ErrModelNotFound
}

// EditFeed owned by user
func EditFeed(id string, feed models.Feed, user models.User) (models.Feed, error) {
	return defaultInstance.EditFeed(id, feed, user)
}

// ChangeFeedCategory changes the category a feed belongs to
func (db *DB) ChangeFeedCategory(feedID, ctgID string, user models.User) error {
	feed := &models.Feed{}
	if db.db.Model(&user).Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return ErrModelNotFound
	}

	prevCtg := &models.Category{
		ID: feed.CategoryID,
	}

	db.db.First(prevCtg)

	db.db.Model(prevCtg).Association("Feeds").Delete(feed)

	newCtg := &models.Category{}
	if db.db.Model(&user).Where("api_id = ?", ctgID).Related(newCtg).RecordNotFound() {
		return ErrModelNotFound
	}

	db.db.Model(newCtg).Association("Feeds").Append(feed)

	return nil
}

// ChangeFeedCategory changes the category a feed belongs to
func ChangeFeedCategory(feedID, ctgID string, user models.User) error {
	return defaultInstance.ChangeFeedCategory(feedID, ctgID, user)
}

// FeedEntries returns all Entries that belong to a feed with feedID
func (db *DB) FeedEntries(feedID string, orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	if marker == models.MarkerNone {
		return nil
	}

	feed := &models.Feed{}
	if db.db.Model(&user).Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return nil
	}

	entries := []models.Entry{}

	query := db.db.Model(&user)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Where("feed_id = ?", feed.ID).Association("Entries").Find(&entries)

	return entries
}

// FeedEntries returns all Entries that belong to a feed with feedID
func FeedEntries(feedID string, orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	return defaultInstance.FeedEntries(feedID, orderByNewest, marker, user)
}

// MarkFeed applies marker to a Feed with id and owned by user
func (db *DB) MarkFeed(id string, marker models.Marker, user models.User) error {
	if feed, found := db.FeedWithAPIID(id, user); found {
		markedEntry := &models.Entry{Mark: marker}
		db.db.Model(markedEntry).Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).Update(markedEntry)
		return nil
	}

	return ErrModelNotFound
}

// MarkFeed applies marker to a Feed with id and owned by user
func MarkFeed(id string, marker models.Marker, user models.User) error {
	return defaultInstance.MarkFeed(id, marker, user)
}

// FeedStats returns all Stats for a Feed with the given id and that is owned by user
func (db *DB) FeedStats(id string, user models.User) models.Stats {
	feed := &models.Feed{}
	if db.db.Model(&user).Where("api_id = ?", id).Related(feed).RecordNotFound() {
		return models.Stats{}
	}

	stats := models.Stats{}

	stats.Unread = db.db.Model(&user).Where("feed_id = ? AND mark = ?", feed.ID, models.MarkerUnread).Association("Entries").Count()
	stats.Read = db.db.Model(&user).Where("feed_id = ? AND mark = ?", feed.ID, models.MarkerRead).Association("Entries").Count()
	stats.Saved = db.db.Model(&user).Where("feed_id = ? AND saved = ?", feed.ID, true).Association("Entries").Count()
	stats.Total = db.db.Model(&user).Where("feed_id = ?", feed.ID).Association("Entries").Count()

	return stats
}

// FeedStats returns all Stats for a Feed with the given id and that is owned by user
func FeedStats(id string, user models.User) models.Stats {
	return defaultInstance.FeedStats(id, user)
}
