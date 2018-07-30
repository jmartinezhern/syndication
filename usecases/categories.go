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
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

type (

	// Category interface defines the Category usescases
	Category interface {
		// New creates a new category. If the category conflicts with an existing category,
		// this errors
		New(name string, user models.User) (models.Category, error)

		// Category returns a category with ID that belongs to user
		Category(id string, user models.User) (models.Category, bool)

		// Categories returns all categories owned by user
		Categories(user models.User) []models.Category

		// Feeds returns all feeds associated to a category
		Feeds(ctgID string, user models.User) ([]models.Feed, error)

		// Edit a category with ID that belongs to user
		Edit(newName, ctgID string, user models.User) (models.Category, error)

		// AddFeeds to a category
		AddFeeds(ctgID string, feeds []string, user models.User)

		// Delete a category with ID that belongs to a user
		Delete(id string, user models.User) error

		// Mark a category
		Mark(id string, marker models.Marker, user models.User) error

		// Entries returns all entries associated to a category
		Entries(id string, order bool, marker models.Marker, user models.User) ([]models.Entry, error)

		// Stats returns statistics on a category items
		Stats(id string, user models.User) (models.Stats, error)
	}

	// CategoryUsecase implements the Category interface
	CategoryUsecase struct {
	}
)

var (
	// ErrCategoryNotFound signals that a category model could not be found
	ErrCategoryNotFound = errors.New("Category not found")

	// ErrCategoryConflicts signals that a category model conflicts with an existing category
	ErrCategoryConflicts = errors.New("Category conflicts")

	// ErrCategoryProtected signals that a modification to a "system" category was attempted
	ErrCategoryProtected = errors.New("Category cannot be modified")
)

// New creates a new category. If the category conflicts with an existing category,
// this errors
func (c *CategoryUsecase) New(name string, user models.User) (models.Category, error) {
	name = strings.ToLower(name)

	if _, found := database.CategoryWithName(name, user); found {
		return models.Category{}, ErrCategoryConflicts
	}

	return database.NewCategory(name, user), nil
}

// Category returns a category with ID that belongs to user
func (c *CategoryUsecase) Category(id string, user models.User) (models.Category, bool) {
	return database.CategoryWithAPIID(id, user)
}

// Categories returns all categories owned by user
func (c *CategoryUsecase) Categories(user models.User) []models.Category {
	return database.Categories(user)
}

// Feeds returns all feeds associated to a category
func (c *CategoryUsecase) Feeds(id string, user models.User) ([]models.Feed, error) {
	if _, found := database.CategoryWithAPIID(id, user); !found {
		return nil, ErrCategoryNotFound
	}

	return database.CategoryFeeds(id, user), nil
}

// Edit a category with ID that belongs to user
func (c *CategoryUsecase) Edit(newName, ctgID string, user models.User) (models.Category, error) {
	ctg, found := database.CategoryWithAPIID(ctgID, user)
	if !found {
		return models.Category{}, ErrCategoryNotFound
	}

	if ctg.Name == models.Uncategorized {
		return models.Category{}, ErrCategoryProtected
	}

	return database.EditCategory(ctgID, models.Category{Name: newName}, user)
}

// AddFeeds to a category
func (c *CategoryUsecase) AddFeeds(ctgID string, feeds []string, user models.User) {
	for _, id := range feeds {
		err := database.ChangeFeedCategory(id, ctgID, user)
		if err != nil {
			log.Error(err)
		}
	}
}

// Delete a category with ID that belongs to a user
func (c *CategoryUsecase) Delete(id string, user models.User) error {
	ctg, found := database.CategoryWithAPIID(id, user)
	if !found {
		return ErrCategoryNotFound
	}

	if ctg.Name == models.Uncategorized {
		return ErrCategoryProtected
	}

	return database.DeleteCategory(id, user)
}

// Mark a category
func (c *CategoryUsecase) Mark(id string, marker models.Marker, user models.User) error {
	err := database.MarkCategory(id, marker, user)
	if err == database.ErrModelNotFound {
		return ErrCategoryNotFound
	}

	return err
}

// Entries returns all entries associated to a category
func (c *CategoryUsecase) Entries(id string, order bool, marker models.Marker, user models.User) ([]models.Entry, error) {
	if _, found := database.CategoryWithAPIID(id, user); !found {
		return nil, ErrCategoryNotFound
	}

	return database.CategoryEntries(id, order, marker, user), nil
}

// Stats returns statistics on a category items
func (c *CategoryUsecase) Stats(id string, user models.User) (models.Stats, error) {
	if _, found := database.CategoryWithAPIID(id, user); !found {
		return models.Stats{}, ErrCategoryNotFound
	}

	return database.CategoryStats(id, user), nil
}
