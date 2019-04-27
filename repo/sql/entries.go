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
	"time"

	"github.com/jinzhu/gorm"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
)

type (
	Entries struct {
		db *DB
	}
)

func NewEntries(db *DB) Entries {
	return Entries{
		db,
	}
}

// Create a new Entry owned by user
func (e Entries) Create(userID string, entry *models.Entry) {
	e.db.db.Model(&models.User{ID: userID}).Association("Entries").Append(entry)

	if entry.Feed.ID != "" {
		var feed models.Feed
		if !e.db.db.Model(&models.User{ID: userID}).Where("id = ?", entry.Feed.ID).Related(&feed).RecordNotFound() {
			e.db.db.Model(&feed).Association("Entries").Append(entry)
		}
	}
}

// EntryWithGUID returns an Entry with GUID and owned by user
func (e Entries) EntryWithGUID(userID, guid string) (entry models.Entry, found bool) {
	found = !e.db.db.Model(&models.User{ID: userID}).Where("guid = ?", guid).Related(&entry).RecordNotFound()
	if found {
		e.db.db.Model(&entry).Related(&entry.Feed)
	}
	return
}

// List all entries owned by user
func (e Entries) List(
	userID, continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	query := e.db.db.Model(&models.User{ID: userID})

	if continuationID != "" {
		entry, found := e.EntryWithID(userID, continuationID)
		if found {
			query = query.Where("created_at >= ?", entry.CreatedAt)
		}
	}

	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Limit(count + 1).Association("Entries").Find(&entries)

	if len(entries) > count {
		next = entries[len(entries)-1].ID
		entries = entries[:len(entries)-1]
	}

	return entries, next
}

// ListFromFeed returns all Entries associated to a feed
func (e Entries) ListFromFeed(
	userID, feedID, continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	var feed models.Feed
	if notFound := e.db.db.Model(&models.User{ID: userID}).Where("id = ?", feedID).Related(&feed).RecordNotFound(); notFound {
		return nil, ""
	}

	query := e.db.db.Model(&feed)

	return e.paginateList(userID, query, continuationID, count, orderByNewest, marker)
}

// ListFromCategory all Entries that are associated to a Category
func (e Entries) ListFromCategory(
	userID, ctgID, continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	var ctg models.Category
	if notFound := e.db.db.Model(&models.User{ID: userID}).Where("id = ?", ctgID).Related(&ctg).RecordNotFound(); notFound {
		return nil, ""
	}

	query := e.db.db.Model(&models.User{ID: userID})

	var feeds []models.Feed
	e.db.db.Model(&ctg).Related(&feeds)
	feedIds := make([]models.ID, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	query.Where("feed_id in (?)", feedIds)

	return e.paginateList(userID, query, continuationID, count, orderByNewest, marker)
}

func (e Entries) paginateList(
	userID string,
	query *gorm.DB,
	continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	if continuationID != "" {
		entry := models.Entry{}
		if !e.db.db.Model(&models.User{ID: userID}).Where("id = ?", continuationID).Related(&entry).RecordNotFound() {
			query = query.Where("created_at >= ?", entry.CreatedAt)
		}
	}

	query.Limit(count + 1).Association("Entries").Find(&entries)

	if len(entries) > count {
		next = entries[len(entries)-1].ID
		entries = entries[:len(entries)-1]
	}

	return entries, next
}

// EntryWithID returns an Entry with id owned by user
func (e Entries) EntryWithID(userID, id string) (entry models.Entry, found bool) {
	found = !e.db.db.Model(&models.User{ID: userID}).Where("id = ?", id).Related(&entry).RecordNotFound()
	return
}

// TagEntries with the given tag for user
func (e Entries) TagEntries(userID, tagID string, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}

	var tag models.Tag
	if e.db.db.Model(&models.User{ID: userID}).Where("id = ?", tagID).Related(&tag).RecordNotFound() {
		return repo.ErrModelNotFound
	}

	entries := make([]models.Entry, len(entryIDs))
	for i, id := range entryIDs {
		entry, found := e.EntryWithID(userID, id)
		if found {
			entries[i] = entry
		}
	}

	e.db.db.Model(&tag).Association("Entries").Append(entries)

	return nil
}

// Mark applies marker to an entry with id and owned by user
func (e Entries) Mark(userID, id string, marker models.Marker) error {
	if entry, found := e.EntryWithID(userID, id); found {
		e.db.db.Model(&entry).Update(&models.Entry{Mark: marker})
		return nil
	}
	return repo.ErrModelNotFound
}

// MarkAll entries
func (e Entries) MarkAll(userID string, marker models.Marker) {
	e.db.db.Model(new(models.Entry)).Where("user_id = ?", userID).Update(models.Entry{Mark: marker})
}

// ListFromTags returns all Entries that are related to a list of tags
func (e Entries) ListFromTags(
	userID string,
	tagIDs []string,
	continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	query := e.db.db.Model(&models.User{ID: userID})

	tagPrimaryKey := func(id models.ID, userID string) models.ID {
		tag := &models.Tag{}
		if e.db.db.Model(&models.User{ID: userID}).Where("id = ?", id).Related(tag).RecordNotFound() {
			return ""
		}
		return tag.ID
	}

	var tagPrimaryKeys []models.ID
	for _, tag := range tagIDs {
		key := tagPrimaryKey(tag, userID)
		if key != "" {
			tagPrimaryKeys = append(tagPrimaryKeys, key)
		}
	}

	sql := "inner join entry_tags ON entry_tags.entry_id = entries.id"

	query.Joins(sql).Where("entry_tags.tag_id in (?)", tagPrimaryKeys)

	return e.paginateList(userID, query, continuationID, count, orderByNewest, marker)
}

// DeleteOldEntries deletes entries older than a timestamp
func (e Entries) DeleteOldEntries(userID string, timestamp time.Time) {
	e.db.db.Delete(models.Entry{}, "user_id = ? AND created_at < ? AND saved = ?", userID, timestamp, false)
}

// Stats returns all Stats for feeds owned by user
func (e Entries) Stats(userID string) models.Stats {
	stats := models.Stats{}

	stats.Unread = e.db.db.Model(&models.User{ID: userID}).Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = e.db.db.Model(&models.User{ID: userID}).Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = e.db.db.Model(&models.User{ID: userID}).Where("saved = ?", true).Association("Entries").Count()
	stats.Total = e.db.db.Model(&models.User{ID: userID}).Association("Entries").Count()

	return stats
}
