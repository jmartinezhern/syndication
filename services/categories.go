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

type (

	// Categories interface defines the Categories service
	Categories interface {
		// New creates a new category. If the category conflicts with an existing category,
		// this errors
		New(name string, user *models.User) (models.Category, error)

		// Category returns a category with ID that belongs to user
		Category(id string, user *models.User) (models.Category, bool)

		// Categories returns a page of categories owned by user and a continuation ID
		Categories(continuationID string, count int, user *models.User) ([]models.Category, string)

		// Feeds returns all feeds associated to a category
		Feeds(id string, continuationID string, count int, user *models.User) ([]models.Feed, string)

		// Uncategorized returns all feeds associated to a category
		Uncategorized(continuationID string, count int, user *models.User) ([]models.Feed, string)

		// Update a category with ID that belongs to user
		Update(newName, ctgID string, user *models.User) (models.Category, error)

		// AddFeeds to a category
		AddFeeds(ctgID string, feeds []string, user *models.User)

		// Delete a category with ID that belongs to a user
		Delete(id string, user *models.User) error

		// Mark a category
		Mark(id string, marker models.Marker, user *models.User) error

		// Entries returns all entries associated to a category
		Entries(id string, continuationID string, count int, order bool, marker models.Marker, user *models.User) ([]models.Entry, string, error)

		// Stats returns statistics on a category items
		Stats(id string, user *models.User) (models.Stats, error)
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
func (c CategoriesService) New(name string, user *models.User) (models.Category, error) {
	name = strings.ToLower(name)

	if _, found := c.ctgsRepo.CategoryWithName(user, name); found {
		return models.Category{}, ErrCategoryConflicts
	}

	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  name,
	}
	c.ctgsRepo.Create(user, &ctg)

	return ctg, nil
}

// Category returns a category with ID that belongs to user
func (c CategoriesService) Category(id string, user *models.User) (models.Category, bool) {
	return c.ctgsRepo.CategoryWithID(user, id)
}

// Categories returns all categories owned by user
func (c CategoriesService) Categories(continuationID string, count int, user *models.User) (categories []models.Category, next string) {
	return c.ctgsRepo.List(user, continuationID, count)
}

// Feeds returns all feeds associated to a category
func (c CategoriesService) Feeds(id, continuationID string, count int, user *models.User) (feeds []models.Feed, next string) {
	return c.ctgsRepo.Feeds(user, id, continuationID, count)
}

// Uncategorized returns all feeds associated to a category
func (c CategoriesService) Uncategorized(continuationID string, count int, user *models.User) (feeds []models.Feed, next string) {
	feeds, next = c.ctgsRepo.Uncategorized(user, continuationID, count)
	return
}

// Update a category with ID that belongs to user
func (c CategoriesService) Update(newName, ctgID string, user *models.User) (models.Category, error) {
	ctg := models.Category{APIID: ctgID, Name: newName}
	err := c.ctgsRepo.Update(user, &ctg)
	if err == repo.ErrModelNotFound {
		return models.Category{}, ErrCategoryNotFound
	}
	return ctg, err
}

// AddFeeds to a category with ctgID
func (c CategoriesService) AddFeeds(ctgID string, feeds []string, user *models.User) {
	for _, id := range feeds {
		err := c.ctgsRepo.AddFeed(user, id, ctgID)
		if err != nil {
			continue
		}
	}
}

// Delete a category with ID that belongs to a user
func (c CategoriesService) Delete(id string, user *models.User) error {
	err := c.ctgsRepo.Delete(user, id)
	if err == repo.ErrModelNotFound {
		return ErrCategoryNotFound
	}
	return err
}

// Mark a category
func (c CategoriesService) Mark(id string, marker models.Marker, user *models.User) error {
	err := c.ctgsRepo.Mark(user, id, marker)
	if err == repo.ErrModelNotFound {
		return ErrCategoryNotFound
	}
	return err
}

// Entries returns all entries associated to a category
func (c CategoriesService) Entries(
	id, continuationID string,
	count int,
	order bool,
	marker models.Marker,
	user *models.User) ([]models.Entry, string, error) {
	if _, found := c.ctgsRepo.CategoryWithID(user, id); !found {
		return nil, "", ErrCategoryNotFound
	}
	entries, next := c.entriesRepo.ListFromCategory(user, id, continuationID, count, order, marker)
	return entries, next, nil
}

// Stats returns statistics on a category items
func (c CategoriesService) Stats(id string, user *models.User) (models.Stats, error) {
	stats, err := c.ctgsRepo.Stats(user, id)
	if err == repo.ErrModelNotFound {
		return models.Stats{}, ErrCategoryNotFound
	} else if err != nil {
		return models.Stats{}, err
	}

	return stats, nil
}
