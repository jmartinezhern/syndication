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

// NewCategory creates a new Category object owned by user
func (db *DB) NewCategory(name string, user models.User) models.Category {
	ctg := models.Category{
		Name:  name,
		APIID: createAPIID(),
	}
	db.db.Model(&user).Association("Categories").Append(&ctg)
	return ctg
}

// NewCategory creates a new Category object owned by user
func NewCategory(name string, user models.User) models.Category {
	return defaultInstance.NewCategory(name, user)
}

// EditCategory owned by user
func (db *DB) EditCategory(id string, newCtg models.Category, user models.User) (models.Category, error) {
	if ctg, found := db.CategoryWithAPIID(id, user); found {
		db.db.Model(&ctg).Updates(newCtg)
		return ctg, nil
	}
	return models.Category{}, ErrModelNotFound
}

// EditCategory owned by user
func EditCategory(id string, ctg models.Category, user models.User) (models.Category, error) {
	return defaultInstance.EditCategory(id, ctg, user)
}

// DeleteCategory with id and owned by user
func (db *DB) DeleteCategory(id string, user models.User) error {
	ctg := &models.Category{}
	if db.db.Model(&user).Where("api_id = ?", id).Related(ctg).RecordNotFound() {
		return ErrModelNotFound
	}

	db.db.Delete(ctg)
	return nil
}

// DeleteCategory with id for user
func DeleteCategory(id string, user models.User) error {
	return defaultInstance.DeleteCategory(id, user)
}

// CategoryWithAPIID returns a category with API ID that belongs to user
func (db *DB) CategoryWithAPIID(id string, user models.User) (ctg models.Category, found bool) {
	found = !db.db.Model(&user).Where("api_id = ?", id).Related(&ctg).RecordNotFound()
	return
}

// CategoryWithAPIID returns a Category with API ID that belongs to user
func CategoryWithAPIID(id string, user models.User) (models.Category, bool) {
	return defaultInstance.CategoryWithAPIID(id, user)
}

// Categories returns a list of all Categories owned by user
func (db *DB) Categories(user models.User) (categories []models.Category) {
	db.db.Model(&user).Association("Categories").Find(&categories)
	return
}

// Categories returns a list of all Categories owned by user
func Categories(user models.User) []models.Category {
	return defaultInstance.Categories(user)
}

// CategoryFeeds returns all Feeds that belong to a category with categoryID
func (db *DB) CategoryFeeds(id string, user models.User) (feeds []models.Feed) {
	if ctg, found := db.CategoryWithAPIID(id, user); found {
		db.db.Model(ctg).Association("Feeds").Find(&feeds)
	}
	return
}

// CategoryFeeds returns all Feeds that belong to a category with categoryID
func CategoryFeeds(id string, user models.User) []models.Feed {
	return defaultInstance.CategoryFeeds(id, user)
}

// CategoryWithName returns a Category that has a matching name and belongs to the given user
func (db *DB) CategoryWithName(name string, user models.User) (ctg models.Category, found bool) {
	found = !db.db.Model(&user).Where("name = ?", name).Related(&ctg).RecordNotFound()
	return
}

// CategoryWithName returns a Category that has a matching name and belongs to the given user
func CategoryWithName(name string, user models.User) (models.Category, bool) {
	return defaultInstance.CategoryWithName(name, user)
}

// CategoryEntries returns all Entries that are related to a Category with id by the entries' owning Feed
func (db *DB) CategoryEntries(id string, orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	if marker == models.MarkerNone {
		return nil
	}

	category := &models.Category{}
	if db.db.Model(&user).Where("api_id = ?", id).Related(category).RecordNotFound() {
		return nil
	}

	var feeds []models.Feed
	var entries []models.Entry

	db.db.Model(category).Related(&feeds)

	query := db.db.Model(&user)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	query.Where("feed_id in (?)", feedIds).Association("Entries").Find(&entries)

	return entries
}

// CategoryEntries returns all Entries that are related to a Category with id by the entries' owning Feed
func CategoryEntries(id string, orderByNewest bool, marker models.Marker, user models.User) []models.Entry {
	return defaultInstance.CategoryEntries(id, orderByNewest, marker, user)
}

// CategoryStats returns all Stats for a Category with the given id and that is owned by user
func (db *DB) CategoryStats(id string, user models.User) models.Stats {
	ctg := &models.Category{}
	if db.db.Model(&user).Where("api_id = ?", id).Related(ctg).RecordNotFound() {
		return models.Stats{}
	}

	var feeds []models.Feed
	db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	query := db.db.Model(&user).Where("feed_id in (?)", feedIds)

	stats := models.Stats{}

	stats.Unread = query.Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = query.Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = query.Where("saved = ?", true).Association("Entries").Count()
	stats.Total = query.Association("Entries").Count()

	return stats
}

// CategoryStats returns all Stats for a Category with the given id and that is owned by user
func CategoryStats(id string, user models.User) models.Stats {
	return defaultInstance.CategoryStats(id, user)
}

// MarkCategory applies marker to a category with id and owned by user
func (db *DB) MarkCategory(id string, marker models.Marker, user models.User) error {
	ctg, found := db.CategoryWithAPIID(id, user)
	if !found {
		return ErrModelNotFound
	}

	var feeds []models.Feed
	db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	markedEntry := &models.Entry{Mark: marker}
	db.db.Model(markedEntry).Where("user_id = ? AND feed_id in (?)", user.ID, feedIds).Update(markedEntry)
	return nil
}

// MarkCategory applies marker to a category with id and owned by user
func MarkCategory(id string, marker models.Marker, user models.User) error {
	return defaultInstance.MarkCategory(id, marker, user)
}
