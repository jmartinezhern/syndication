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
		AdminWithID(id string) (models.Admin, bool)
		InitialUser() (admin models.Admin, found bool)
		Delete(id string) error
		OwnsKey(key *models.APIKey, admin *models.Admin) bool
		AddAPIKey(key *models.APIKey, admin *models.Admin)
		AdminWithName(name string) (models.Admin, bool)
	}

	Categories interface {
		Create(user *models.User, ctg *models.Category)
		Update(user *models.User, ctg *models.Category) error
		Delete(user *models.User, id string) error
		CategoryWithID(user *models.User, id string) (models.Category, bool)
		CategoryWithName(user *models.User, name string) (models.Category, bool)
		List(user *models.User, continuationID string, count int) ([]models.Category, string)
		Feeds(user *models.User, ctgID string, continuationID string, count int) ([]models.Feed, string)
		Uncategorized(user *models.User, continuationID string, count int) ([]models.Feed, string)
		AddFeed(user *models.User, feedID, ctgID string) error
		Stats(user *models.User, ctgID string) (models.Stats, error)
		Mark(user *models.User, ctgID string, marker models.Marker) error
	}

	Users interface {
		Create(user *models.User)
		Update(user *models.User) error
		UserWithID(id string) (models.User, bool)
		Delete(id string) error
		OwnsKey(key *models.APIKey, user *models.User) bool
		AddAPIKey(key *models.APIKey, user *models.User)
		List(continuationID string, count int) ([]models.User, string)
		UserWithName(name string) (models.User, bool)
	}

	Entries interface {
		Create(user *models.User, entry *models.Entry)
		EntryWithID(user *models.User, id string) (models.Entry, bool)
		EntryWithGUID(user *models.User, guid string) (models.Entry, bool)
		List(
			user *models.User,
			continuationID string,
			count int,
			orderByNewest bool,
			marker models.Marker) ([]models.Entry, string)
		ListFromTags(
			user *models.User,
			tagIDs []string,
			continuationID string,
			count int,
			orderByNewest bool,
			marker models.Marker) ([]models.Entry, string)
		ListFromCategory(
			user *models.User,
			ctgID, continuationID string,
			count int,
			orderByNewest bool,
			marker models.Marker) ([]models.Entry, string)
		ListFromFeed(
			user *models.User,
			feedID, continuationID string,
			count int,
			orderByNewest bool,
			marked models.Marker) ([]models.Entry, string)
		TagEntries(user *models.User, tagID string, entryIDs []string) error
		Mark(user *models.User, id string, marker models.Marker) error
		MarkAll(user *models.User, marker models.Marker)
		Stats(user *models.User) models.Stats
	}

	Feeds interface {
		Create(user *models.User, feed *models.Feed)
		Update(user *models.User, feed *models.Feed) error
		Delete(user *models.User, id string) error
		FeedWithID(user *models.User, id string) (models.Feed, bool)
		List(user *models.User, continuationID string, count int) ([]models.Feed, string)
		Mark(user *models.User, id string, marker models.Marker) error
		Stats(user *models.User, ctgID string) (models.Stats, error)
	}

	Tags interface {
		Create(user *models.User, tag *models.Tag)
		Update(user *models.User, tag *models.Tag) error
		Delete(user *models.User, id string) error
		TagWithID(user *models.User, id string) (models.Tag, bool)
		TagWithName(user *models.User, name string) (models.Tag, bool)
		List(user *models.User, continuationID string, count int) ([]models.Tag, string)
	}
)
