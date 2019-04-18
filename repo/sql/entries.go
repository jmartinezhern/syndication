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
func (e Entries) Create(user *models.User, entry *models.Entry) {
	e.db.db.Model(user).Association("Entries").Append(entry)

	if entry.Feed.APIID != "" {
		var feed models.Feed
		if !e.db.db.Model(user).Where("api_id = ?", entry.Feed.APIID).Related(&feed).RecordNotFound() {
			e.db.db.Model(&feed).Association("Entries").Append(entry)
		}
	}
}

// EntryWithGUID returns an Entry with GUID and owned by user
func (e Entries) EntryWithGUID(user *models.User, guid string) (entry models.Entry, found bool) {
	found = !e.db.db.Model(user).Where("guid = ?", guid).Related(&entry).RecordNotFound()
	if found {
		e.db.db.Model(&entry).Related(&entry.Feed)
	}
	return
}

// List all entries owned by user
func (e Entries) List(
	user *models.User,
	continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	query := e.db.db.Model(user)

	if continuationID != "" {
		entry, found := e.EntryWithID(user, continuationID)
		if found {
			query = query.Where("id >= ?", entry.ID)
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
		next = entries[len(entries)-1].APIID
		entries = entries[:len(entries)-1]
	}

	return entries, next
}

// ListFromFeed returns all Entries associated to a feed
func (e Entries) ListFromFeed(
	user *models.User,
	feedID, continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	var feed models.Feed
	if notFound := e.db.db.Model(user).Where("api_id = ?", feedID).Related(&feed).RecordNotFound(); notFound {
		return nil, ""
	}

	query := e.db.db.Model(&feed)

	return e.paginateList(user, query, continuationID, count, orderByNewest, marker)
}

// ListFromCategory all Entries that are associated to a Category
func (e Entries) ListFromCategory(
	user *models.User,
	ctgID,
	continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	var ctg models.Category
	if notFound := e.db.db.Model(user).Where("api_id = ?", ctgID).Related(&ctg).RecordNotFound(); notFound {
		return nil, ""
	}

	query := e.db.db.Model(user)

	var feeds []models.Feed
	e.db.db.Model(&ctg).Related(&feeds)
	feedIds := make([]uint, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	query.Where("feed_id in (?)", feedIds)

	return e.paginateList(user, query, continuationID, count, orderByNewest, marker)
}

func (e Entries) paginateList(
	user *models.User,
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
		if !e.db.db.Model(user).Where("api_id = ?", continuationID).Related(&entry).RecordNotFound() {
			query = query.Where("id >= ?", entry.ID)
		}
	}

	query.Limit(count + 1).Association("Entries").Find(&entries)

	if len(entries) > count {
		next = entries[len(entries)-1].APIID
		entries = entries[:len(entries)-1]
	}

	return entries, next
}

// EntryWithID returns an Entry with id owned by user
func (e Entries) EntryWithID(user *models.User, id string) (entry models.Entry, found bool) {
	found = !e.db.db.Model(user).Where("api_id = ?", id).Related(&entry).RecordNotFound()
	return
}

// TagEntries with the given tag for user
func (e Entries) TagEntries(user *models.User, tagID string, entryIDs []string) error {
	if len(entryIDs) == 0 {
		return nil
	}

	var tag models.Tag
	if e.db.db.Model(user).Where("api_id = ?", tagID).Related(&tag).RecordNotFound() {
		return repo.ErrModelNotFound
	}

	entries := make([]models.Entry, len(entryIDs))
	for i, id := range entryIDs {
		entry, found := e.EntryWithID(user, id)
		if found {
			entries[i] = entry
		}
	}

	e.db.db.Model(&tag).Association("Entries").Append(entries)

	return nil
}

// Mark applies marker to an entry with id and owned by user
func (e Entries) Mark(user *models.User, id string, marker models.Marker) error {
	if entry, found := e.EntryWithID(user, id); found {
		e.db.db.Model(&entry).Update(&models.Entry{Mark: marker})
		return nil
	}
	return repo.ErrModelNotFound
}

// MarkAll entries
func (e Entries) MarkAll(user *models.User, marker models.Marker) {
	e.db.db.Model(new(models.Entry)).Where("user_id = ?", user.ID).Update(models.Entry{Mark: marker})
}

// ListFromTags returns all Entries that are related to a list of tags
func (e Entries) ListFromTags(
	user *models.User,
	tagIDs []string,
	continuationID string,
	count int,
	orderByNewest bool,
	marker models.Marker) (entries []models.Entry, next string) {
	query := e.db.db.Model(user)

	tagPrimaryKey := func(apiID string, user *models.User) uint {
		tag := &models.Tag{}
		if e.db.db.Model(user).Where("api_id = ?", apiID).Related(tag).RecordNotFound() {
			return 0
		}
		return tag.ID
	}

	var tagPrimaryKeys []uint
	for _, tag := range tagIDs {
		key := tagPrimaryKey(tag, user)
		if key != 0 {
			tagPrimaryKeys = append(tagPrimaryKeys, key)
		}
	}

	sql := "inner join entry_tags ON entry_tags.entry_id = entries.id"

	query.Joins(sql).Where("entry_tags.tag_id in (?)", tagPrimaryKeys)

	return e.paginateList(user, query, continuationID, count, orderByNewest, marker)
}

// DeleteOldEntries deletes entries older than a timestamp
func (e Entries) DeleteOldEntries(user *models.User, timestamp time.Time) {
	e.db.db.Delete(models.Entry{}, "user_id = ? AND created_at < ? AND saved = ?", user.ID, timestamp, false)
}

// Stats returns all Stats for feeds owned by user
func (e Entries) Stats(user *models.User) models.Stats {
	stats := models.Stats{}

	stats.Unread = e.db.db.Model(user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = e.db.db.Model(user).Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = e.db.db.Model(user).Where("saved = ?", true).Association("Entries").Count()
	stats.Total = e.db.db.Model(user).Association("Entries").Count()

	return stats
}
