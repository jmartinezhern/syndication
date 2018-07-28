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
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

type (
	// Entry interface defines the Entry usescases
	Entry interface {
		// Entry returns an entry with id that belongs to user
		Entry(id string, user models.User) (models.Entry, error)

		// Entries returns all entries belong to a user with a marker
		Entries(order bool, marker models.Marker, user models.User) []models.Entry

		// Mark entry with id
		Mark(id string, marker models.Marker, user models.User) error

		// Stats returns statistics for all entries
		Stats(user models.User) models.Stats
	}

	// EntryUsecase implements Entry usecase
	EntryUsecase struct {
	}
)

var (
	// ErrEntryNotFound signals that an entry model could not be found
	ErrEntryNotFound = errors.New("Entry not found")
)

// Entry returns an entry with ID that belongs to user
func (e *EntryUsecase) Entry(id string, user models.User) (models.Entry, error) {
	entry, found := database.EntryWithAPIID(id, user)
	if !found {
		return models.Entry{}, ErrEntryNotFound
	}

	return entry, nil
}

// Entries returns all entries belong to a user with a marker
func (e *EntryUsecase) Entries(order bool, marker models.Marker, user models.User) []models.Entry {
	return database.Entries(order, marker, user)
}

// Mark entry with id
func (e *EntryUsecase) Mark(id string, marker models.Marker, user models.User) error {
	err := database.MarkEntry(id, marker, user)
	if err == database.ErrModelNotFound {
		return ErrEntryNotFound
	}

	return err
}

// Stats returns statistics for all entries
func (e *EntryUsecase) Stats(user models.User) models.Stats {
	return database.Stats(user)
}
