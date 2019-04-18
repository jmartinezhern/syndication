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
	Categories struct {
		db *DB
	}
)

func NewCategories(db *DB) Categories {
	return Categories{
		db,
	}
}

// Create a new Category owned by user
func (c Categories) Create(user *models.User, ctg *models.Category) {
	c.db.db.Model(user).Association("Categories").Append(ctg)
}

// Update a category owned by user
func (c Categories) Update(user *models.User, ctg *models.Category) error {
	dbCtg, found := c.CategoryWithID(user, ctg.APIID)
	if !found {
		return repo.ErrModelNotFound
	}

	c.db.db.Model(&dbCtg).Updates(ctg)
	return nil
}

// Delete a category with id owned by user
func (c Categories) Delete(user *models.User, id string) error {
	ctg, found := c.CategoryWithID(user, id)
	if !found {
		return repo.ErrModelNotFound
	}

	c.db.db.Delete(ctg)
	return nil
}

// CategoryWithID returns a category with ID owned by user
func (c Categories) CategoryWithID(user *models.User, id string) (ctg models.Category, found bool) {
	found = !c.db.db.Model(user).Where("api_id = ?", id).Related(&ctg).RecordNotFound()
	return
}

// List all Categories owned by user
func (c Categories) List(user *models.User, continuationID string, count int) (categories []models.Category, next string) {
	query := c.db.db.Model(user)

	if continuationID != "" {
		if ctg, found := c.CategoryWithID(user, continuationID); found {
			query = query.Where("id >= ?", ctg.ID)
		}
	}

	query.Limit(count + 1).Association("Categories").Find(&categories)

	if len(categories) > count {
		next = categories[len(categories)-1].APIID
		categories = categories[:len(categories)-1]
	}

	return
}

// Feeds returns all Feeds in category with ctgID owned by user
func (c Categories) Feeds(user *models.User, ctgID, continuationID string, count int) (feeds []models.Feed, next string) {
	ctg, found := c.CategoryWithID(user, ctgID)
	if !found {
		return nil, ""
	}

	query := c.db.db.Model(&ctg)

	if continuationID != "" {
		feed := models.Feed{}
		if !c.db.db.Model(user).Where("api_id = ?", continuationID).Related(&feed).RecordNotFound() {
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

// Uncategorized returns all Feeds that belong to a category with categoryID
func (c Categories) Uncategorized(user *models.User, continuationID string, count int) (feeds []models.Feed, next string) {
	query := c.db.db.Model(user).Where("category_id = ?", 0)

	if continuationID != "" {
		feed := models.Feed{}
		if !c.db.db.Model(user).Where("api_id = ?", continuationID).Related(&feed).RecordNotFound() {
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

// CategoryWithName returns a Category that has a matching name and belongs to the given user
func (c Categories) CategoryWithName(user *models.User, name string) (ctg models.Category, found bool) {
	found = !c.db.db.Model(user).Where("name = ?", name).Related(&ctg).RecordNotFound()
	return
}

// AddFeed associates a feed to a category with ctgID
func (c Categories) AddFeed(user *models.User, feedID, ctgID string) error {
	var feed models.Feed
	if c.db.db.Model(user).Where("api_id = ?", feedID).Related(&feed).RecordNotFound() {
		return repo.ErrModelNotFound
	}

	ctg, found := c.CategoryWithID(user, ctgID)
	if !found {
		return repo.ErrModelNotFound
	}

	return c.db.db.Model(&ctg).Association("Feeds").Replace(&feed).Error
}

// Stats returns all Stats for a Category with the given id and that is owned by user
func (c Categories) Stats(user *models.User, ctgID string) (models.Stats, error) {
	ctg, found := c.CategoryWithID(user, ctgID)
	if !found {
		return models.Stats{}, repo.ErrModelNotFound
	}

	var feeds []models.Feed
	c.db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	query := c.db.db.Model(user).Where("feed_id in (?)", feedIds)

	stats := models.Stats{}

	stats.Unread = query.Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = query.Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = query.Where("saved = ?", true).Association("Entries").Count()
	stats.Total = query.Association("Entries").Count()

	return stats, nil
}

// Mark applies marker to a category with id and owned by user
func (c Categories) Mark(user *models.User, ctgID string, marker models.Marker) error {
	ctg, found := c.CategoryWithID(user, ctgID)
	if !found {
		return repo.ErrModelNotFound
	}

	var feeds []models.Feed
	c.db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	markedEntry := &models.Entry{Mark: marker}
	c.db.db.Model(markedEntry).Where("user_id = ? AND feed_id in (?)", user.ID, feedIds).Update(markedEntry)
	return nil
}
