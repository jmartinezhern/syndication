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
)

type (
	// Entries interface defines the Entries service
	Entries interface {
		// Entry returns an entry with id that belongs to user
		Entry(id string, user *models.User) (models.Entry, error)

		// Entries returns all entries belong to a user with a marker
		Entries(continuationID string, count int, order bool, marker models.Marker, user *models.User) ([]models.Entry, string)

		// Mark entry with id
		Mark(id string, marker models.Marker, user *models.User) error

		// MarkAll entries
		MarkAll(marker models.Marker, user *models.User)

		// Stats returns statistics for all entries
		Stats(user *models.User) models.Stats
	}

	// EntriesService implements Entries service
	EntriesService struct {
		repo repo.Entries
	}
)

var (
	// ErrEntryNotFound signals that an entry model could not be found
	ErrEntryNotFound = errors.New("entry not found")
)

func NewEntriesService(entriesRepo repo.Entries) EntriesService {
	return EntriesService{
		entriesRepo,
	}
}

// Entry returns an entry with ID that belongs to user
func (e EntriesService) Entry(id string, user *models.User) (models.Entry, error) {
	entry, found := e.repo.EntryWithID(user, id)
	if !found {
		return models.Entry{}, ErrEntryNotFound
	}

	return entry, nil
}

// Entries returns all entries belong to a user with a marker
func (e EntriesService) Entries(
	continuationID string,
	count int,
	order bool,
	marker models.Marker,
	user *models.User) (entries []models.Entry, next string) {
	return e.repo.List(user, continuationID, count, order, marker)
}

// Mark entry with id
func (e EntriesService) Mark(id string, marker models.Marker, user *models.User) error {
	err := e.repo.Mark(user, id, marker)
	if err == repo.ErrModelNotFound {
		return ErrEntryNotFound
	}

	return err
}

// MarkAll entries
func (e EntriesService) MarkAll(marker models.Marker, user *models.User) {
	e.repo.MarkAll(user, marker)
}

// Stats returns statistics for all entries
func (e EntriesService) Stats(user *models.User) models.Stats {
	return e.repo.Stats(user)
}
