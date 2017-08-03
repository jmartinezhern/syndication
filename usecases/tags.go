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

package usecases

import (
	"errors"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	// Tag defines the Tag Usecase interface
	Tag interface {
		// New creates a new tag
		New(name string, user models.User) (models.Tag, error)

		// Tags returns all tags owned by user
		Tags(user models.User) []models.Tag

		// Delete a tag id
		Delete(id string, user models.User) error

		// Edit a tag with id
		Edit(id string, newTag models.Tag, user models.User) (models.Tag, error)

		// Apply associates a tag with an entry
		Apply(id string, entries []string, user models.User) error

		// Tag returns a tag with id
		Tag(id string, user models.User) (models.Tag, bool)

		// Entries returns all entries associated with a tag with id
		Entries(id string, marker models.Marker, order bool, user models.User) ([]models.Entry, error)
	}

	// TagUsecase implementation
	TagUsecase struct{}
)

var (
	// ErrTagNotFound signals that a tag could not be found
	ErrTagNotFound = errors.New("Tag not found")

	// ErrTagConflicts signals that a tag conflicts with an existing tag
	ErrTagConflicts = errors.New("Model conflicts")
)

// New creates a new tag
func (t *TagUsecase) New(name string, user models.User) (models.Tag, error) {
	if _, found := database.TagWithName(name, user); found {
		return models.Tag{}, ErrTagConflicts
	}

	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  name,
	}
	database.CreateTag(&tag, user)

	return tag, nil
}

// Tags returns all tags owned by user
func (t *TagUsecase) Tags(user models.User) []models.Tag {
	return database.Tags(user)
}

// Delete a tag id
func (t *TagUsecase) Delete(id string, user models.User) error {
	err := database.DeleteTag(id, user)
	if err == database.ErrModelNotFound {
		return ErrTagNotFound
	}
	return err
}

// Edit a tag with id
func (t *TagUsecase) Edit(id string, newTag models.Tag, user models.User) (models.Tag, error) {
	mdfTag, err := database.EditTag(id, newTag, user)
	if err == database.ErrModelNotFound {
		return models.Tag{}, ErrTagNotFound
	}

	return mdfTag, err
}

// Apply associates a tag with an entry
func (t *TagUsecase) Apply(id string, entries []string, user models.User) error {
	err := database.TagEntries(id, entries, user)
	if err == database.ErrModelNotFound {
		return ErrTagNotFound
	}

	return err
}

// Tag returns a tag with id
func (t *TagUsecase) Tag(id string, user models.User) (models.Tag, bool) {
	return database.TagWithAPIID(id, user)
}

// Entries returns all entries associated with a tag with id
func (t *TagUsecase) Entries(id string, marker models.Marker, order bool, user models.User) ([]models.Entry, error) {
	if _, found := database.TagWithAPIID(id, user); !found {
		return nil, ErrTagNotFound
	}
	return database.EntriesFromTag(id, marker, order, user), nil
}
