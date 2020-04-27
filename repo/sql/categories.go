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
	"github.com/jinzhu/gorm"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
)

type (
	Categories struct {
		db *gorm.DB
	}
)

func NewCategories(db *gorm.DB) Categories {
	return Categories{
		db,
	}
}

// Create a new Category owned by user
func (c Categories) Create(userID string, ctg *models.Category) {
	c.db.Model(&models.User{ID: userID}).Association("Categories").Append(ctg)
}

// Update a category owned by user
func (c Categories) Update(userID string, ctg *models.Category) error {
	dbCtg, found := c.CategoryWithID(userID, ctg.ID)
	if !found {
		return repo.ErrModelNotFound
	}

	c.db.Model(&dbCtg).Updates(ctg)

	return nil
}

// Delete a category with id owned by user
func (c Categories) Delete(userID, id string) error {
	ctg, found := c.CategoryWithID(userID, id)
	if !found {
		return repo.ErrModelNotFound
	}

	c.db.Delete(ctg)

	return nil
}

// CategoryWithID returns a category with ID owned by user
func (c Categories) CategoryWithID(userID, id string) (ctg models.Category, found bool) {
	found = !c.db.Model(&models.User{ID: userID}).Where("id = ?", id).Related(&ctg).RecordNotFound()
	return
}

// List all Categories owned by user
func (c Categories) List(userID string, page models.Page) (categories []models.Category, next string) {
	query := c.db.Model(&models.User{ID: userID})

	if page.ContinuationID != "" {
		if ctg, found := c.CategoryWithID(userID, page.ContinuationID); found {
			query = query.Where("created_at >= ?", ctg.CreatedAt)
		}
	}

	query.Limit(page.Count + 1).Association("Categories").Find(&categories)

	if len(categories) > page.Count {
		next = categories[len(categories)-1].ID
		categories = categories[:len(categories)-1]
	}

	return
}

// Feeds returns all Feeds in category with ctgID owned by user
func (c Categories) Feeds(userID string, page models.Page) (feeds []models.Feed, next string) {
	ctg, found := c.CategoryWithID(userID, page.FilterID)
	if !found {
		return nil, ""
	}

	query := c.db.Model(&ctg)

	if page.ContinuationID != "" {
		feed := models.Feed{}
		if !c.db.Model(&models.User{
			ID: userID,
		}).Where("id = ?", page.ContinuationID).Related(&feed).RecordNotFound() {
			query = query.Where("created_at >= ?", feed.CreatedAt)
		}
	}

	query.Limit(page.Count + 1).Association("Feeds").Find(&feeds)

	if len(feeds) > page.Count {
		next = feeds[len(feeds)-1].ID
		feeds = feeds[:len(feeds)-1]
	}

	return
}

// Uncategorized returns all Feeds that belong to a category with categoryID
func (c Categories) Uncategorized(userID string, page models.Page) (feeds []models.Feed, next string) {
	query := c.db.Model(&models.User{ID: userID}).Where("category_id = ?", "")

	if page.ContinuationID != "" {
		feed := models.Feed{}
		if !c.db.Model(&models.User{
			ID: userID,
		}).Where("id = ?", page.ContinuationID).Related(&feed).RecordNotFound() {
			query = query.Where("created_at >= ?", feed.CreatedAt)
		}
	}

	query.Limit(page.Count + 1).Association("Feeds").Find(&feeds)

	if len(feeds) > page.Count {
		next = feeds[len(feeds)-1].ID
		feeds = feeds[:len(feeds)-1]
	}

	return
}

// CategoryWithName returns a Category that has a matching name and belongs to the given user
func (c Categories) CategoryWithName(userID, name string) (ctg models.Category, found bool) {
	found = !c.db.Model(&models.User{ID: userID}).Where("name = ?", name).Related(&ctg).RecordNotFound()
	return
}

// AddFeed associates a feed to a category with ctgID
func (c Categories) AddFeed(userID, feedID, ctgID string) error {
	var feed models.Feed
	if c.db.Model(&models.User{ID: userID}).Where("id = ?", feedID).Related(&feed).RecordNotFound() {
		return repo.ErrModelNotFound
	}

	ctg, found := c.CategoryWithID(userID, ctgID)
	if !found {
		return repo.ErrModelNotFound
	}

	return c.db.Model(&ctg).Association("Feeds").Replace(&feed).Error
}

// Stats returns all Stats for a Category with the given id and that is owned by user
func (c Categories) Stats(userID, ctgID string) (models.Stats, error) {
	ctg, found := c.CategoryWithID(userID, ctgID)
	if !found {
		return models.Stats{}, repo.ErrModelNotFound
	}

	var feeds []models.Feed

	c.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]string, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	query := c.db.Model(&models.User{ID: userID}).Where("feed_id in (?)", feedIds)

	stats := models.Stats{}

	stats.Unread = query.Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = query.Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = query.Where("saved = ?", true).Association("Entries").Count()
	stats.Total = query.Association("Entries").Count()

	return stats, nil
}

// Mark applies marker to a category with id and owned by user
func (c Categories) Mark(userID, ctgID string, marker models.Marker) error {
	ctg, found := c.CategoryWithID(userID, ctgID)
	if !found {
		return repo.ErrModelNotFound
	}

	var feeds []models.Feed

	c.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]models.ID, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	markedEntry := &models.Entry{Mark: marker}
	c.db.Model(markedEntry).Where("user_id = ? AND feed_id in (?)", userID, feedIds).Update(markedEntry)

	return nil
}
