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

package repo

import (
	"errors"

	"github.com/jmartinezhern/syndication/models"
)

var (
	// ErrModelNotFound signals that an operation was attempted
	// on a model that is not found in the repo.
	ErrModelNotFound = errors.New("model not found")
)

type (
	Admins interface {
		Create(admin *models.Admin)
		Update(id string, admin *models.Admin) error
		AdminWithName(name string) (models.Admin, bool)
		AdminWithID(id string) (models.Admin, bool)
		List(continuationID string, count int) ([]models.Admin, string)
		Delete(id string) error
	}

	Categories interface {
		Create(userID string, ctg *models.Category)
		Update(userID string, ctg *models.Category) error
		Delete(userID, id string) error
		CategoryWithID(userID, id string) (models.Category, bool)
		CategoryWithName(userID, name string) (models.Category, bool)
		List(userID, continuationID string, count int) ([]models.Category, string)
		Feeds(userID, ctgID, continuationID string, count int) ([]models.Feed, string)
		Uncategorized(userID, continuationID string, count int) ([]models.Feed, string)
		AddFeed(userID, feedID, ctgID string) error
		Stats(userID, ctgID string) (models.Stats, error)
		Mark(userID, ctgID string, marker models.Marker) error
	}

	Users interface {
		Create(user *models.User)
		Update(user *models.User) error
		UserWithName(name string) (models.User, bool)
		UserWithID(id string) (models.User, bool)
		Delete(id string) error
		List(continuationID string, count int) ([]models.User, string)
	}

	Entries interface {
		Create(userID string, entry *models.Entry)
		EntryWithID(userID, id string) (models.Entry, bool)
		EntryWithGUID(userID, guid string) (models.Entry, bool)
		List(userID string, page models.Page) ([]models.Entry, string)
		ListFromTags(userID string, tagIDs []string, page models.Page) ([]models.Entry, string)
		ListFromCategory(userID, ctgID string, page models.Page) ([]models.Entry, string)
		ListFromFeed(userID, feedID string, page models.Page) ([]models.Entry, string)
		TagEntries(userID, tagID string, entryIDs []string) error
		Mark(userID, id string, marker models.Marker) error
		MarkAll(userID string, marker models.Marker)
		Stats(userID string) models.Stats
	}

	Feeds interface {
		Create(userID string, feed *models.Feed)
		Update(userID string, feed *models.Feed) error
		Delete(userID, id string) error
		FeedWithID(userID, id string) (models.Feed, bool)
		List(userID, continuationID string, count int) ([]models.Feed, string)
		Mark(userID, id string, marker models.Marker) error
		Stats(userID, ctgID string) (models.Stats, error)
	}

	Tags interface {
		Create(userID string, tag *models.Tag)
		Update(userID string, tag *models.Tag) error
		Delete(userID, id string) error
		TagWithID(userID, id string) (models.Tag, bool)
		TagWithName(userID, name string) (models.Tag, bool)
		List(userID, continuationID string, count int) ([]models.Tag, string)
	}
)
