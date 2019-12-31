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
	"database/sql"
	log "github.com/sirupsen/logrus"
	"time"

	sq "github.com/Masterminds/squirrel"

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
func (c Categories) Create(userID string, ctg *models.Category) {
	ctg.CreatedAt = time.Now()
	ctg.UpdatedAt = time.Now()

	stmnt, args, err := sq.Insert("categories").
		Columns("id", "name", "user_id", "created_at", "updated_at").
		Values(ctg.ID, ctg.Name, userID, ctg.CreatedAt, ctg.UpdatedAt).
		ToSql()
	if err != nil {
		panic(err)
	}

	_, err = c.db.db.DB().Exec(stmnt, args...)
	if err != nil {
		// TODO - return
		panic(err)
	}

	// TODO - remove
	ctg.UserID = userID
}

// Update a category owned by user
func (c Categories) Update(userID string, ctg *models.Category) error {
	ctg.UpdatedAt = time.Now()

	stmnt, args, err := sq.Update("categories").
		Set("name", ctg.Name).
		Set("updated_at", ctg.UpdatedAt).
		Where("id = ? AND user_id = ?", ctg.ID, userID).
		ToSql()
	if err != nil {
		panic(err)
	}

	res, err := c.db.db.DB().Exec(stmnt, args...)
	if err != nil {
		return err
	}

	if count, _ := res.RowsAffected(); count == 0 {
		return repo.ErrModelNotFound
	}

	return nil
}

// Delete a category with id owned by user
func (c Categories) Delete(userID, id string) error {
	stmnt, args, err := sq.Delete("categories").
		Where("id = ? AND user_id = ?", id, userID).
		ToSql()
	if err != nil {
		panic(err)
	}

	res, err := c.db.db.DB().Exec(stmnt, args...)
	if err != nil {
		return err
	}

	if count, _ := res.RowsAffected(); count == 0 {
		return repo.ErrModelNotFound
	}

	return nil
}

// CategoryWithID returns a category with ID owned by user
func (c Categories) CategoryWithID(userID, id string) (models.Category, bool) {
	stmnt, args, err := sq.Select("id", "created_at", "updated_at", "name").
		From("categories").
		Where("id = ? AND user_id = ?", id, userID).
		ToSql()
	if err != nil {
		panic(err)
	}

	row := c.db.db.DB().QueryRow(stmnt, args...)

	var ctg models.Category
	if err = row.Scan(&ctg.ID, &ctg.CreatedAt, &ctg.UpdatedAt, &ctg.Name); err != nil {
		if err != sql.ErrNoRows {
			log.Error(err)
		}

		return models.Category{}, false
	}

	return ctg, true
}

// List all Categories owned by user
func (c Categories) List(userID string, page models.Page) ([]models.Category, string) {
	query := sq.Select("id, name, created_at, updated_at").
		From("categories").
		Where("user_id = ?", userID)

	if page.ContinuationID != "" {
		if ctg, found := c.CategoryWithID(userID, page.ContinuationID); found {
			query = query.Where("created_at >= ?", ctg.CreatedAt)
		}
	}

	stmnt, args, err := query.Limit(uint64(page.Count + 1)).ToSql()
	if err != nil {
		panic(err)
	}

	rows, err := c.db.db.DB().Query(stmnt, args...)
	if err != nil {
		log.Error(err)
		return nil, ""
	}

	var categories []models.Category

	for rows.Next() {
		var ctg models.Category
		err := rows.Scan(&ctg.ID, &ctg.Name, &ctg.CreatedAt, &ctg.UpdatedAt)
		if err == sql.ErrNoRows {
			return nil, ""
		} else if err != nil {
			log.Error(err)
			return nil, ""
		}
		categories = append(categories, ctg)
	}

	if categories != nil && len(categories) > page.Count {
		return categories[:len(categories)-1], categories[len(categories)-1].ID
	}

	return categories, ""
}

// Feeds returns all Feeds in category with ctgID owned by user
func (c Categories) Feeds(userID string, page models.Page) (feeds []models.Feed, next string) {
	ctg, found := c.CategoryWithID(userID, page.FilterID)
	if !found {
		return nil, ""
	}

	query := c.db.db.Model(&ctg)

	if page.ContinuationID != "" {
		feed := models.Feed{}
		if !c.db.db.Model(&models.User{
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
	query := c.db.db.Model(&models.User{ID: userID}).Where("category_id = ?", "")

	if page.ContinuationID != "" {
		feed := models.Feed{}
		if !c.db.db.Model(&models.User{
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
	found = !c.db.db.Model(&models.User{ID: userID}).Where("name = ?", name).Related(&ctg).RecordNotFound()
	return
}

// AddFeed associates a feed to a category with ctgID
func (c Categories) AddFeed(userID, feedID, ctgID string) error {
	var feed models.Feed
	if c.db.db.Model(&models.User{ID: userID}).Where("id = ?", feedID).Related(&feed).RecordNotFound() {
		return repo.ErrModelNotFound
	}

	ctg, found := c.CategoryWithID(userID, ctgID)
	if !found {
		return repo.ErrModelNotFound
	}

	return c.db.db.Model(&ctg).Association("Feeds").Replace(&feed).Error
}

// Stats returns all Stats for a Category with the given id and that is owned by user
func (c Categories) Stats(userID, ctgID string) (models.Stats, error) {
	ctg, found := c.CategoryWithID(userID, ctgID)
	if !found {
		return models.Stats{}, repo.ErrModelNotFound
	}

	var feeds []models.Feed

	c.db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]string, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	query := c.db.db.Model(&models.User{ID: userID}).Where("feed_id in (?)", feedIds)

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

	c.db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]models.ID, len(feeds))
	for idx := range feeds {
		feedIds[idx] = feeds[idx].ID
	}

	markedEntry := &models.Entry{Mark: marker}
	c.db.db.Model(markedEntry).Where("user_id = ? AND feed_id in (?)", userID, feedIds).Update(markedEntry)

	return nil
}
