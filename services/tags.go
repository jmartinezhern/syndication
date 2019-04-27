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
		New(name string, userID string) (models.Tag, error)

		// List returns all tags owned by user
		List(continuationID string, count int, userID string) ([]models.Tag, string)

		// Delete a tag id
		Delete(id string, userID string) error

		// Update a tag with id
		Update(id string, newName string, userID string) (models.Tag, error)

		// Apply associates a tag with an entry
		Apply(id string, entries []string, userID string) error

		// Tag returns a tag with id
		Tag(id string, userID string) (models.Tag, bool)

		// Entries returns all entries associated with a tag with id
		Entries(id string, continuationID string, count int, order bool, marker models.Marker, userID string) ([]models.Entry, string)
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
func (t TagsService) New(name, userID string) (models.Tag, error) {
	if _, found := t.tagsRepo.TagWithName(userID, name); found {
		return models.Tag{}, ErrTagConflicts
	}

	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: name,
	}
	t.tagsRepo.Create(userID, &tag)

	return tag, nil
}

// List returns all tags owned by user
func (t TagsService) List(continuationID string, count int, userID string) (tags []models.Tag, next string) {
	return t.tagsRepo.List(userID, continuationID, count)
}

// Delete a tag id
func (t TagsService) Delete(id, userID string) error {
	err := t.tagsRepo.Delete(userID, id)
	if err == repo.ErrModelNotFound {
		return ErrTagNotFound
	}
	return err
}

// Update a tag with id
func (t TagsService) Update(id, newName, userID string) (models.Tag, error) {
	if _, found := t.tagsRepo.TagWithName(userID, newName); found {
		return models.Tag{}, ErrTagConflicts
	}

	tag := models.Tag{ID: id, Name: newName}
	err := t.tagsRepo.Update(userID, &tag)
	if err == repo.ErrModelNotFound {
		return models.Tag{}, ErrTagNotFound
	}

	return tag, err
}

// Apply associates a tag with an entry
func (t TagsService) Apply(id string, entries []string, userID string) error {
	err := t.entriesRepo.TagEntries(userID, id, entries)
	if err == repo.ErrModelNotFound {
		return ErrTagNotFound
	}

	return err
}

// Tag returns a tag with id
func (t TagsService) Tag(id, userID string) (models.Tag, bool) {
	return t.tagsRepo.TagWithID(userID, id)
}

// Entries returns all entries associated with a tag with id
func (t TagsService) Entries(
	id, continuationID string,
	count int,
	order bool,
	marker models.Marker,
	userID string) (entries []models.Entry, next string) {
	return t.entriesRepo.ListFromTags(userID, []string{id}, continuationID, count, order, marker)
}
