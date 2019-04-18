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

package services

import (
	"errors"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	// Tag defines the Tag service interface
	Tag interface {
		// New creates a new tag
		New(name string, user *models.User) (models.Tag, error)

		// List returns all tags owned by user
		List(continuationID string, count int, user *models.User) ([]models.Tag, string)

		// Delete a tag id
		Delete(id string, user *models.User) error

		// Update a tag with id
		Update(id string, newName string, user *models.User) (models.Tag, error)

		// Apply associates a tag with an entry
		Apply(id string, entries []string, user *models.User) error

		// Tag returns a tag with id
		Tag(id string, user *models.User) (models.Tag, bool)

		// Entries returns all entries associated with a tag with id
		Entries(id string, continuationID string, count int, order bool, marker models.Marker, user *models.User) ([]models.Entry, string)
	}

	// TagsService implementation
	TagsService struct {
		tagsRepo    repo.Tags
		entriesRepo repo.Entries
	}
)

var (
	// ErrTagNotFound signals that a tag could not be found
	ErrTagNotFound = errors.New("tag not found")

	// ErrTagConflicts signals that a tag conflicts with an existing tag
	ErrTagConflicts = errors.New("model conflicts")
)

func NewTagsService(tagsRepo repo.Tags, entriesRepo repo.Entries) TagsService {
	return TagsService{
		tagsRepo,
		entriesRepo,
	}
}

// New creates a new tag
func (t TagsService) New(name string, user *models.User) (models.Tag, error) {
	if _, found := t.tagsRepo.TagWithName(user, name); found {
		return models.Tag{}, ErrTagConflicts
	}

	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  name,
	}
	t.tagsRepo.Create(user, &tag)

	return tag, nil
}

// List returns all tags owned by user
func (t TagsService) List(continuationID string, count int, user *models.User) (tags []models.Tag, next string) {
	return t.tagsRepo.List(user, continuationID, count)
}

// Delete a tag id
func (t TagsService) Delete(id string, user *models.User) error {
	err := t.tagsRepo.Delete(user, id)
	if err == repo.ErrModelNotFound {
		return ErrTagNotFound
	}
	return err
}

// Update a tag with id
func (t TagsService) Update(id, newName string, user *models.User) (models.Tag, error) {
	if _, found := t.tagsRepo.TagWithName(user, newName); found {
		return models.Tag{}, ErrTagConflicts
	}

	tag := models.Tag{APIID: id, Name: newName}
	err := t.tagsRepo.Update(user, &tag)
	if err == repo.ErrModelNotFound {
		return models.Tag{}, ErrTagNotFound
	}

	return tag, err
}

// Apply associates a tag with an entry
func (t TagsService) Apply(id string, entries []string, user *models.User) error {
	err := t.entriesRepo.TagEntries(user, id, entries)
	if err == repo.ErrModelNotFound {
		return ErrTagNotFound
	}

	return err
}

// Tag returns a tag with id
func (t TagsService) Tag(id string, user *models.User) (models.Tag, bool) {
	return t.tagsRepo.TagWithID(user, id)
}

// Entries returns all entries associated with a tag with id
func (t TagsService) Entries(
	id, continuationID string,
	count int,
	order bool,
	marker models.Marker,
	user *models.User) (entries []models.Entry, next string) {
	return t.entriesRepo.ListFromTags(user, []string{id}, continuationID, count, order, marker)
}
