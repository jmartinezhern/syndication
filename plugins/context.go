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

package plugins

import (
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

type (
	// APICtx collects information and resources available for the API Plugin
	APICtx struct {
		User *UserCtx
	}

	// UserCtx allows access to a single user's data
	UserCtx struct {
		db   *database.DB
		user *models.User
	}
)

// NewUserCtx creates a new user from a database
func NewUserCtx(db *database.DB, user *models.User) UserCtx {
	return UserCtx{db, user}
}

// HasUser checks if the API Ctx also has a UserCtx
func (c APICtx) HasUser() bool {
	return c.User != nil
}

// Entries retrives all entries from a user with an order and marker
func (c UserCtx) Entries(orderByNewest bool, marker models.Marker) []models.Entry {
	return c.db.Entries(orderByNewest, marker, c.user)
}

// EntriesFromCategory retrieves entries in a category from a user
func (c UserCtx) EntriesFromCategory(categoryID string, orderByNewest bool, marker models.Marker) []models.Entry {
	return c.db.EntriesFromCategory(categoryID, orderByNewest, marker, c.user)
}

// EntriesFromFeed retrieves entries belonging to a feed owned by a user
func (c UserCtx) EntriesFromFeed(feedID string, orderByNewest bool, marker models.Marker) []models.Entry {
	return c.db.EntriesFromFeed(feedID, orderByNewest, marker, c.user)
}

// EntriesFromTag retrieves all entries related to the given tag which is owned by a user
func (c UserCtx) EntriesFromTag(tagID string, orderByNewest bool, marker models.Marker) []models.Entry {
	return c.db.EntriesFromTag(tagID, marker, orderByNewest, c.user)
}

// EntriesFromMultipleTags retrieves all entries related to the given tags which are owned by a user
func (c UserCtx) EntriesFromMultipleTags(tagIDs []string, orderByNewest bool, marker models.Marker) []models.Entry {
	return c.db.EntriesFromMultipleTags(tagIDs, orderByNewest, marker, c.user)
}

// EntryWithAPIID retrieves a single entry
func (c UserCtx) EntryWithAPIID(id string) (models.Entry, bool) {
	return c.db.EntryWithAPIID(id, c.user)
}

// Feeds retrieves all feeds belonging to a user
func (c UserCtx) Feeds() []models.Feed {
	return c.db.Feeds(c.user)
}

// FeedsFromCategory retrieves feeds contained in category that is owned by a user
func (c UserCtx) FeedsFromCategory(categoryID string) []models.Feed {
	return c.db.FeedsFromCategory(categoryID, c.user)
}

// FeedWithAPIID retrieves a single feed with the given ID
func (c UserCtx) FeedWithAPIID(id string) (models.Feed, bool) {
	return c.db.FeedWithAPIID(id, c.user)
}

// DeleteFeed deletes a feed owned by a user
func (c UserCtx) DeleteFeed(id string) error {
	return c.db.DeleteFeed(id, c.user)
}

// EditFeed modifies writable proprieties of a feed owned by a user
func (c UserCtx) EditFeed(feed *models.Feed) error {
	return c.db.EditFeed(feed, c.user)
}

// Categories retrieves all categories owned by a user
func (c UserCtx) Categories() []models.Category {
	return c.db.Categories(c.user)
}

// CategoryWithAPIID retrieves a single category with the given ID
func (c UserCtx) CategoryWithAPIID(id string) (models.Category, bool) {
	return c.db.CategoryWithAPIID(id, c.user)
}

// EditCategory modifies writable properties of a category owned by a user
func (c UserCtx) EditCategory(ctg *models.Category) error {
	return c.db.EditCategory(ctg, c.user)
}

// DeleteCategory deletes a category owned by a user
func (c UserCtx) DeleteCategory(id string) error {
	return c.db.DeleteCategory(id, c.user)
}

// ChangeFeedCategory moves a feed from its current category to a different one.
func (c UserCtx) ChangeFeedCategory(feedID, ctgID string) error {
	return c.db.ChangeFeedCategory(feedID, ctgID, c.user)
}

// Tags retrieves all tags owned by a user
func (c UserCtx) Tags() []models.Tag {
	return c.db.Tags(c.user)
}

// TagWithAPIID returns a tag with the give ID
func (c UserCtx) TagWithAPIID(id string) (models.Tag, bool) {
	return c.db.TagWithAPIID(id, c.user)
}

// EditTagName updates the name of a tag owned by a user
func (c UserCtx) EditTagName(tagID, name string, user *models.User) error {
	return c.db.EditTagName(tagID, name, c.user)
}

// DeleteTag deletes a tag owned by a user
func (c UserCtx) DeleteTag(id string) error {
	return c.db.DeleteTag(id, c.user)
}

// TagEntries applies a tag to multiple entries
func (c UserCtx) TagEntries(tagID string, entries []string) error {
	return c.db.TagEntries(tagID, entries, c.user)
}

// CategoryStats retrieves statistics related to a category
func (c UserCtx) CategoryStats(id string) models.Stats {
	return c.db.CategoryStats(id, c.user)
}

// FeedStats retrieves statistics related to a feed.
func (c UserCtx) FeedStats(id string) models.Stats {
	return c.db.FeedStats(id, c.user)
}

// Stats retrieves stats about all resources owned by a user
func (c UserCtx) Stats() models.Stats {
	return c.db.Stats(c.user)
}

// MarkFeed applies a marker to a feed and its entries.
func (c UserCtx) MarkFeed(id string, marker models.Marker) error {
	return c.db.MarkFeed(id, marker, c.user)
}

// MarkCategory applies a marker to a category's feeds
func (c UserCtx) MarkCategory(id string, marker models.Marker) error {
	return c.db.MarkCategory(id, marker, c.user)
}

// MarkEntry applies a marker to an entry
func (c UserCtx) MarkEntry(id string, marker models.Marker) error {
	return c.db.MarkEntry(id, marker, c.user)
}
