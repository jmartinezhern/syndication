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

package sql

import (
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
)

type (
	Feeds struct {
		db *DB
	}
)

func NewFeeds(db *DB) Feeds {
	return Feeds{
		db,
	}
}

// Create a new feed owned by user
func (f Feeds) Create(userID string, feed *models.Feed) {
	f.db.db.Model(&models.User{ID: userID}).Association("Feeds").Append(feed)

	if feed.Category.ID != "" {
		var ctg models.Category
		found := !f.db.db.Model(&models.User{ID: userID}).Where("id = ?", feed.Category.ID).Related(&ctg).RecordNotFound()
		if found {
			f.db.db.Model(&ctg).Association("Feeds").Append(feed)
		}
	}
}

// Update a feed owned by user
func (f Feeds) Update(userID string, feed *models.Feed) error {
	dbFeed, found := f.FeedWithID(userID, feed.ID)
	if !found {
		return repo.ErrModelNotFound
	}

	f.db.db.Model(&dbFeed).Updates(feed)
	return nil
}

// Delete a feed owned by user
func (f Feeds) Delete(userID, id string) error {
	feed, found := f.FeedWithID(userID, id)
	if !found {
		return repo.ErrModelNotFound
	}

	f.db.db.Delete(&feed)
	return nil
}

// FeedWithID returns a Feed with id and owned by user
func (f Feeds) FeedWithID(userID, id string) (feed models.Feed, found bool) {
	found = !f.db.db.Model(&models.User{ID: userID}).Where("id = ?", id).Related(&feed).RecordNotFound()
	if found {
		f.db.db.Model(&feed).Related(&feed.Category)
	}
	return
}

// List all Feeds owned by user
func (f Feeds) List(userID, continuationID string, count int) (feeds []models.Feed, next string) {
	query := f.db.db.Model(&models.User{ID: userID})

	if continuationID != "" {
		if feed, found := f.FeedWithID(userID, continuationID); found {
			query = query.Where("created_at >= ?", feed.CreatedAt)
		}
	}

	query.Limit(count + 1).Association("Feeds").Find(&feeds)

	if len(feeds) > count {
		next = feeds[len(feeds)-1].ID
		feeds = feeds[:len(feeds)-1]
	}

	return
}

// Mark applies marker to a Feed with id and owned by user
func (f Feeds) Mark(userID, id string, marker models.Marker) error {
	if feed, found := f.FeedWithID(userID, id); found {
		entry := models.Entry{Mark: marker}
		f.db.db.Model(&entry).Where("user_id = ? AND feed_id = ?", userID, feed.ID).Update(&entry)
		return nil
	}

	return repo.ErrModelNotFound
}

// Stats returns all Stats for a Feed with the given id and that is owned by user
func (f Feeds) Stats(userID, id string) (models.Stats, error) {
	feed, found := f.FeedWithID(userID, id)
	if !found {
		return models.Stats{}, repo.ErrModelNotFound
	}

	query := f.db.db.Model(&feed)

	stats := models.Stats{}

	stats.Unread = query.Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = query.Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = query.Where("saved = ?", true).Association("Entries").Count()
	stats.Total = query.Association("Entries").Count()

	return stats, nil
}
