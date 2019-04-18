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
	// Feed defines the Feed service interface
	Feed interface {
		// New creates a new Feed
		New(title, subscription string, ctgID string, user *models.User) (models.Feed, error)

		// Feeds returns all feeds owned by user
		Feeds(continuationID string, count int, user *models.User) ([]models.Feed, string)

		// Feed returns a feed with id owned by user
		Feed(id string, user *models.User) (models.Feed, bool)

		// Update feed owned by user
		Update(feed *models.Feed, user *models.User) error

		// Delete a feed with id
		Delete(id string, user *models.User) error

		// Mark a feed with id
		Mark(id string, marker models.Marker, user *models.User) error

		// Entries returns all entry items associated to a feed
		Entries(id string, continuationID string, count int, order bool, marker models.Marker, user *models.User) ([]models.Entry, string)

		// Stats returns statistics of a feed
		Stats(id string, user *models.User) (models.Stats, error)
	}

	// FeedService implementation
	FeedService struct {
		feedsRepo   repo.Feeds
		ctgsRepo    repo.Categories
		entriesRepo repo.Entries
	}
)

var (
	// ErrFeedCategoryNotFound signals that a request category for a given feed
	// was not found
	ErrFeedCategoryNotFound = errors.New("cannot find requested category for feed")

	// ErrFeedNotFound signals that a feed could not be found
	ErrFeedNotFound = errors.New("feed not found")

	// ErrFetchingFeed Signals that an error occurred while fetching
	// a RSS or Atom feed
	ErrFetchingFeed = errors.New("could not fetch feed")
)

func NewFeedsService(feedsRepo repo.Feeds, ctgsRepo repo.Categories, entriesRepo repo.Entries) FeedService {
	return FeedService{
		feedsRepo,
		ctgsRepo,
		entriesRepo,
	}
}

// New creates a new Feed
func (f FeedService) New(title, subscription, ctgID string, user *models.User) (models.Feed, error) {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Subscription: subscription,
	}

	if ctgID != "" {
		ctg, found := f.ctgsRepo.CategoryWithID(user, ctgID)
		if !found {
			return models.Feed{}, ErrFeedCategoryNotFound
		}
		feed.Category = ctg
	}

	f.feedsRepo.Create(user, &feed)

	fetchedFeed, entries, err := utils.PullFeed(subscription, "")
	if err != nil {
		return models.Feed{}, ErrFetchingFeed
	}

	if feed.Title != "" {
		fetchedFeed.Title = feed.Title
	}
	fetchedFeed.APIID = feed.APIID

	err = f.feedsRepo.Update(user, &fetchedFeed)
	if err == repo.ErrModelNotFound {
		return models.Feed{}, ErrFeedNotFound
	} else if err != nil {
		return models.Feed{}, err
	}

	for idx := range entries {
		entry := entries[idx]
		entry.APIID = utils.CreateAPIID()
		entry.Feed = feed
		f.entriesRepo.Create(user, &entry)
	}

	return fetchedFeed, nil
}

// Feeds returns all feeds owned by user
func (f FeedService) Feeds(continuationID string, count int, user *models.User) (feeds []models.Feed, next string) {
	return f.feedsRepo.List(user, continuationID, count)
}

// Feed returns a feed with id owned by user
func (f FeedService) Feed(id string, user *models.User) (models.Feed, bool) {
	return f.feedsRepo.FeedWithID(user, id)
}

// Update a feed owned by user
func (f FeedService) Update(feed *models.Feed, user *models.User) error {
	err := f.feedsRepo.Update(user, feed)
	if err == repo.ErrModelNotFound {
		return ErrFeedNotFound
	}
	return err
}

// Delete a feed with id
func (f FeedService) Delete(id string, user *models.User) error {
	err := f.feedsRepo.Delete(user, id)
	if err == repo.ErrModelNotFound {
		return ErrFeedNotFound
	}

	return err
}

// Mark a feed with id
func (f FeedService) Mark(id string, marker models.Marker, user *models.User) error {
	err := f.feedsRepo.Mark(user, id, marker)
	if err == repo.ErrModelNotFound {
		return ErrFeedNotFound
	}

	return err
}

// Entries returns all entry items associated to a feed
func (f FeedService) Entries(
	id, continuationID string,
	count int,
	order bool,
	marker models.Marker,
	user *models.User) (entries []models.Entry, next string) {
	return f.entriesRepo.ListFromFeed(user, id, continuationID, count, order, marker)
}

// Stats returns statistics of a feed
func (f FeedService) Stats(id string, user *models.User) (models.Stats, error) {
	stats, err := f.feedsRepo.Stats(user, id)
	if err == repo.ErrModelNotFound {
		return stats, ErrFeedNotFound
	} else if err != nil {
		return stats, err
	}

	return stats, nil
}
