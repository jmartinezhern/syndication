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

	log "github.com/sirupsen/logrus"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/sync"
)

type (
	// Feed defines the Feed Usecase interface
	Feed interface {
		// New creates a new Feed
		New(title, subscription string, user models.User) models.Feed

		// Feeds returns all feeds owned by user
		Feeds(user models.User) []models.Feed

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
	FeedUsecase struct {
	}
)

var (
	// ErrFeedNotFound signals that a feed could not be found
	ErrFeedNotFound = errors.New("Feed not found")
)

// New creates a new Feed
func (f *FeedUsecase) New(title, subscription string, user models.User) models.Feed {
	feed := database.NewFeed(title, subscription, user)
	fetchedFeed, entries, err := sync.PullFeed(subscription, "")
	if err != nil {
		// Consume the error for now
		log.Error(err)
		return feed
	}

	feed, err = database.EditFeed(feed.APIID, fetchedFeed, user)
	if err != nil {
		log.Error(err)
		return feed
	}

	_, err = database.NewEntries(entries, feed.APIID, user)
	if err != nil {
		log.Error(err)
	}
	return feed
}

// Feeds returns all feeds owned by user
func (f *FeedUsecase) Feeds(user models.User) []models.Feed {
	return database.Feeds(user)
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
