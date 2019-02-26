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
	"github.com/jmartinezhern/syndication/sync"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	// Feed defines the Feed Usecase interface
	Feed interface {
		// New creates a new Feed
		New(title, subscription string, ctgID string, user models.User) (models.Feed, error)

		// Feeds returns all feeds owned by user
		Feeds(continuationID string, count int, user models.User) ([]models.Feed, string)

		// Feed returns a feed with id owned by user
		Feed(id string, user models.User) (models.Feed, bool)

		// Edit feed with id
		Edit(id string, newFeed models.Feed, user models.User) (models.Feed, error)

		// Delete a feed with id
		Delete(id string, user models.User) error

		// Mark a feed with id
		Mark(id string, marker models.Marker, user models.User) error

		// Entries returns all entry items associated to a feed
		Entries(id string, order bool, marker models.Marker, user models.User) ([]models.Entry, error)

		// Stats returns statistics of a feed
		Stats(id string, user models.User) (models.Stats, error)
	}

	// FeedUsecase implementation
	FeedUsecase struct{}
)

var (
	// ErrFeedCategoryNotFound signals that a request category for a given feed
	// was not found
	ErrFeedCategoryNotFound = errors.New("Cannot find requested category for feed")

	// ErrFeedNotFound signals that a feed could not be found
	ErrFeedNotFound = errors.New("Feed not found")

	// ErrFetchingFeed Signals that an error ocurred while fetching
	// a RSS or Atom feed
	ErrFetchingFeed = errors.New("Could not fetch feed")
)

// New creates a new Feed
func (f *FeedUsecase) New(title, subscription string, ctgID string, user models.User) (models.Feed, error) {
	if ctgID == "" {
		ctg, found := database.CategoryWithName(models.Uncategorized, user)
		if !found {
			panic("System category could not be found")
		}

		ctgID = ctg.APIID
	} else {
		_, found := database.CategoryWithAPIID(ctgID, user)
		if !found {
			return models.Feed{}, ErrFeedCategoryNotFound
		}
	}

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Subscription: subscription,
	}
	err := database.CreateFeed(&feed, ctgID, user)
	if err != nil {
		return models.Feed{}, err
	}

	fetchedFeed, entries, err := sync.PullFeed(subscription, "")
	if err != nil {
		return models.Feed{}, ErrFetchingFeed
	}

	feed, err = database.EditFeed(feed.APIID, fetchedFeed, user)
	if err != nil {
		return models.Feed{}, err
	}

	_, err = database.NewEntries(entries, feed.APIID, user)
	if err != nil {
		return models.Feed{}, err
	}

	return feed, nil
}

// Feeds returns all feeds owned by user
func (f *FeedUsecase) Feeds(continuationID string, count int, user models.User) ([]models.Feed, string) {
	return database.Feeds(continuationID, count, user)
}

// Feed returns a feed with id owned by user
func (f *FeedUsecase) Feed(id string, user models.User) (models.Feed, bool) {
	return database.FeedWithAPIID(id, user)
}

// Edit feed with id
func (f *FeedUsecase) Edit(id string, newFeed models.Feed, user models.User) (models.Feed, error) {
	feed, err := database.EditFeed(id, newFeed, user)
	if err == database.ErrModelNotFound {
		return models.Feed{}, ErrFeedNotFound
	}
	return feed, err
}

// Delete a feed with id
func (f *FeedUsecase) Delete(id string, user models.User) error {
	err := database.DeleteFeed(id, user)
	if err == database.ErrModelNotFound {
		return ErrFeedNotFound
	}

	return err
}

// Mark a feed with id
func (f *FeedUsecase) Mark(id string, marker models.Marker, user models.User) error {
	err := database.MarkFeed(id, marker, user)
	if err == database.ErrModelNotFound {
		return ErrFeedNotFound
	}

	return err
}

// Entries returns all entry items associated to a feed
func (f *FeedUsecase) Entries(id string, order bool, marker models.Marker, user models.User) ([]models.Entry, error) {
	if _, found := database.FeedWithAPIID(id, user); !found {
		return nil, ErrFeedNotFound
	}

	return database.FeedEntries(id, order, marker, user), nil
}

// Stats returns statistics of a feed
func (f *FeedUsecase) Stats(id string, user models.User) (models.Stats, error) {
	if _, found := database.FeedWithAPIID(id, user); !found {
		return models.Stats{}, ErrFeedNotFound
	}
	return database.FeedStats(id, user), nil
}
