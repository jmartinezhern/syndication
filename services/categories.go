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
	"strings"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

//go:generate mockgen -source=categories.go -destination=categories_mock.go -package=services

type (
	// Categories interface defines the Categories service
	Categories interface {
		// New creates a new category. If the category conflicts with an existing category,
		// this errors
		New(userID, name string) (models.Category, error)

		// Category returns a category with ID that belongs to user
		Category(userID, id string) (models.Category, bool)

		// Categories returns a page of categories owned by user
		Categories(userID string, page models.Page) ([]models.Category, string)

		// Feeds returns all feeds associated to a category
		Feeds(userID string, page models.Page) ([]models.Feed, string)

		// Uncategorized returns all feeds associated to a category
		Uncategorized(userID string, page models.Page) ([]models.Feed, string)

		// Update a category with ID that belongs to user
		Update(userID, ctgID, newName string) (models.Category, error)

		// AddFeeds to a category
		AddFeeds(userID, ctgID string, feeds []string)

		// Delete a category with ID that belongs to a user
		Delete(userID, id string) error

		// Mark a category
		Mark(userID, id string, marker models.Marker) error

		// Entries returns all entries associated to a category
		Entries(userID string, page models.Page) ([]models.Entry, string, error)

		// Stats returns statistics on a category items
		Stats(userID, id string) (models.Stats, error)
	}

	// CategoriesService implements the Categories interface
	CategoriesService struct {
		ctgsRepo    repo.Categories
		entriesRepo repo.Entries
	}
)

var (
	// ErrCategoryNotFound signals that a category model could not be found
	ErrCategoryNotFound = errors.New("categories not found")

	// ErrCategoryConflicts signals that a category model conflicts with an existing category
	ErrCategoryConflicts = errors.New("categories conflicts")
)

func NewCategoriesService(ctgsRepo repo.Categories, entriesRepo repo.Entries) CategoriesService {
	return CategoriesService{
		ctgsRepo,
		entriesRepo,
	}
}

// New creates a new category. If the category conflicts with an existing category,
// this errors
func (c CategoriesService) New(userID, name string) (models.Category, error) {
	name = strings.ToLower(name)

	if _, found := c.ctgsRepo.CategoryWithName(userID, name); found {
		return models.Category{}, ErrCategoryConflicts
	}

	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: name,
	}
	c.ctgsRepo.Create(userID, &ctg)

	return ctg, nil
}

// Category returns a category with ID that belongs to user
func (c CategoriesService) Category(userID, id string) (models.Category, bool) {
	return c.ctgsRepo.CategoryWithID(userID, id)
}

// Categories returns all categories owned by user
func (c CategoriesService) Categories(userID string, page models.Page) (categories []models.Category, next string) {
	return c.ctgsRepo.List(userID, page)
}

// Feeds returns all feeds associated to a category
func (c CategoriesService) Feeds(userID string, page models.Page) (feeds []models.Feed, next string) {
	return c.ctgsRepo.Feeds(userID, page)
}

// Uncategorized returns all feeds associated to a category
func (c CategoriesService) Uncategorized(userID string, page models.Page) (feeds []models.Feed, next string) {
	feeds, next = c.ctgsRepo.Uncategorized(userID, page)
	return
}

// Update a category with ID that belongs to user
func (c CategoriesService) Update(userID, ctgID, newName string) (models.Category, error) {
	ctg := models.Category{ID: ctgID, Name: newName}

	err := c.ctgsRepo.Update(userID, &ctg)
	if err == repo.ErrModelNotFound {
		return models.Category{}, ErrCategoryNotFound
	}

	return ctg, err
}

// AddFeeds to a category with ctgID
func (c CategoriesService) AddFeeds(userID, ctgID string, feeds []string) {
	for _, id := range feeds {
		err := c.ctgsRepo.AddFeed(userID, id, ctgID)
		if err != nil {
			continue
		}
	}
}

// Delete a category with ID that belongs to a user
func (c CategoriesService) Delete(userID, id string) error {
	err := c.ctgsRepo.Delete(userID, id)
	if err == repo.ErrModelNotFound {
		return ErrCategoryNotFound
	}

	return err
}

// Mark a category
func (c CategoriesService) Mark(userID, id string, marker models.Marker) error {
	err := c.ctgsRepo.Mark(userID, id, marker)
	if err == repo.ErrModelNotFound {
		return ErrCategoryNotFound
	}

	return err
}

// Entries returns all entries associated to a category
func (c CategoriesService) Entries(userID string, page models.Page) ([]models.Entry, string, error) {
	if _, found := c.ctgsRepo.CategoryWithID(userID, page.FilterID); !found {
		return nil, "", ErrCategoryNotFound
	}

	entries, next := c.entriesRepo.ListFromCategory(userID, page)

	return entries, next, nil
}

// Stats returns statistics on a category items
func (c CategoriesService) Stats(userID, id string) (models.Stats, error) {
	stats, err := c.ctgsRepo.Stats(userID, id)
	if err == repo.ErrModelNotFound {
		return models.Stats{}, ErrCategoryNotFound
	} else if err != nil {
		return models.Stats{}, err
	}

	return stats, nil
}
