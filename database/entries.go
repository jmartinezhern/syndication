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

import "github.com/varddum/syndication/models"

// NewEntry creates a new Entry object owned by user
func (db *DB) NewEntry(entry models.Entry, feedID string, user models.User) (models.Entry, error) {
	feed, found := db.FeedWithAPIID(feedID, user)
	if !found {
		return models.Entry{}, ErrModelNotFound
	}

	entry.APIID = createAPIID()
	entry.Feed = feed
	entry.FeedID = feed.ID

	db.db.Model(&user).Association("Entries").Append(entry)
	db.db.Model(&feed).Association("Entries").Append(entry)

	return entry, nil
}

// NewEntry creates a new Entry object owned by user
func NewEntry(entry models.Entry, feedID string, user models.User) (models.Entry, error) {
	return defaultInstance.NewEntry(entry, feedID, user)
}

// NewEntries creates multiple new Entry objects which
// are all owned by feed with feedAPIID and user
func (db *DB) NewEntries(entries []models.Entry, feedID string, user models.User) ([]models.Entry, error) {
	if len(entries) == 0 {
		// Nothing to do
		return nil, nil
	}

	feed, found := db.FeedWithAPIID(feedID, user)
	if !found {
		return nil, ErrModelNotFound
	}

	for i, entry := range entries {
		entry.APIID = createAPIID()

		db.db.Model(&user).Association("Entries").Append(&entry)
		db.db.Model(&feed).Association("Entries").Append(&entry)

		entries[i] = entry
	}

	return entries, nil
}

// NewEntries creates multiple new Entry objects which
// are all owned by feed with feedAPIID and user
func NewEntries(entries []models.Entry, feedID string, user models.User) ([]models.Entry, error) {
	return defaultInstance.NewEntries(entries, feedID, user)
}

// EntryWithGUIDExists returns true if an Entry exists with the given guid and is owned by user
func (db *DB) EntryWithGUIDExists(guid string, feedID string, user models.User) bool {
	userModel := db.db.Model(&user)
	feed := new(models.Feed)
	if userModel.Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return false
	}

	return !userModel.Where("guid = ? AND feed_id = ?", guid, feed.ID).Related(&models.Entry{}).RecordNotFound()
}

// EntryWithGUIDExists returns true if an Entry exists with the given guid and is owned by user
func EntryWithGUIDExists(guid string, feedID string, user models.User) bool {
	return defaultInstance.EntryWithGUIDExists(guid, feedID, user)
}

// Entries returns a list of all entries owned by user
func (db *DB) Entries(orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	entries := []models.Entry{}
	if marker == models.MarkerNone {
		return nil
	}

	query := db.db.Model(&user)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Association("Entries").Find(&entries)

	return entries
}

// Entries returns a list of all entries owned by user
func Entries(orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	return defaultInstance.Entries(orderByNewest, marker, user)
}

// EntryWithAPIID returns an Entry with id that belongs to user
func (db *DB) EntryWithAPIID(apiID string, user models.User) (entry models.Entry, found bool) {
	found = !db.db.Model(&user).Where("api_id = ?", apiID).Related(&entry).RecordNotFound()
	return
}

// EntryWithAPIID returns an Entry with id that belongs to user
func EntryWithAPIID(apiID string, user models.User) (models.Entry, bool) {
	return defaultInstance.EntryWithAPIID(apiID, user)
}

// TagEntries with the given tag for user
func (db *DB) TagEntries(tagID string, entries []string, user models.User) error {
	if len(entries) == 0 {
		return nil
	}

	tag, found := db.TagWithAPIID(tagID, user)
	if !found {
		return ErrModelNotFound
	}

	dbEntries := make([]models.Entry, len(entries))
	for i, entry := range entries {
		dbEntry, found := db.EntryWithAPIID(entry, user)
		if found {
			dbEntries[i] = dbEntry
		}
	}

	for _, entry := range dbEntries {
		db.db.Model(tag).Association("Entries").Append(&entry)
	}

	return nil
}

// TagEntries with the given tag for user
func TagEntries(tagID string, entries []string, user models.User) error {
	return defaultInstance.TagEntries(tagID, entries, user)
}

// MarkEntry applies marker to an entry with id and owned by user
func (db *DB) MarkEntry(id string, marker models.Marker, user models.User) error {
	if entry, found := db.EntryWithAPIID(id, user); found {
		db.db.Model(&entry).Update(models.Entry{Mark: marker})
		return nil
	}
	return ErrModelNotFound
}

// MarkEntry applies marker to an entry with id and owned by user
func MarkEntry(id string, marker models.Marker, user models.User) error {
	return defaultInstance.MarkEntry(id, marker, user)
}

// MarkAll entries
func (db *DB) MarkAll(marker models.Marker, user models.User) {
	db.db.Model(new(models.Entry)).Where("user_id = ?", user.ID).Update(models.Entry{Mark: marker})
}

// MarkAll entries
func MarkAll(marker models.Marker, user models.User) {
	defaultInstance.MarkAll(marker, user)
}

// EntriesFromTag returns all Entries which are tagged with tagID
func (db *DB) EntriesFromTag(tagID string, marker models.Marker, orderByNewest bool, user models.User) []models.Entry {
	tag := &models.Tag{}
	if db.db.Model(&user).Where("api_id = ?", tagID).Related(tag).RecordNotFound() {
		return nil
	}

	query := db.db.Model(tag)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	var entries []models.Entry

	query.Association("Entries").Find(&entries)

	return entries
}

// EntriesFromTag returns all Entries which are tagged with tagID
func EntriesFromTag(tagID string, marker models.Marker, orderByNewest bool, user models.User) []models.Entry {
	return defaultInstance.EntriesFromTag(tagID, marker, orderByNewest, user)
}

// EntriesFromMultipleTags returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func (db *DB) EntriesFromMultipleTags(tagIDs []string, orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	order := db.db.Model(&user).Select("entries.title")
	if orderByNewest {
		order = order.Order("created_at DESC")
	} else {
		order = order.Order("created_at ASC")
	}

	if marker != models.MarkerAny {
		order = order.Where("mark = ?", marker)
	}

	var tagPrimaryKeys []uint
	for _, tag := range tagIDs {
		key := db.tagPrimaryKey(tag, user)
		if key != 0 {
			tagPrimaryKeys = append(tagPrimaryKeys, key)
		}
	}

	var entries []models.Entry

	query := "inner join entry_tags ON entry_tags.entry_id = entries.id"
	order.Joins(query).Where("entry_tags.tag_id in (?)", tagPrimaryKeys).Related(&entries)

	return entries
}

// EntriesFromMultipleTags returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func EntriesFromMultipleTags(tagIDs []string, orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	return defaultInstance.EntriesFromMultipleTags(tagIDs, orderByNewest, marker, user)
}
