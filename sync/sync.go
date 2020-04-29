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

package sync

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

const (
	maxThreads           = 10
	maxPageSizePerThread = 100
)

// Service defines properties for running a Feed Sync Service.
// Service will update all feeds for all users periodically.
type Service struct {
	ticker *time.Ticker

	quit      chan bool
	userQueue chan models.User

	wg sync.WaitGroup

	interval time.Duration

	usersRepo   repo.Users
	feedsRepo   repo.Feeds
	entriesRepo repo.Entries
}

func (s *Service) syncUserHandler() {
	defer s.wg.Done()

	for {
		user, ok := <-s.userQueue
		if !ok {
			return
		}

		s.SyncUser(user.ID)
	}
}

func (s *Service) updateFeed(userID string, feed *models.Feed) {
	if !time.Now().After(feed.LastUpdated.Add(s.interval)) {
		return
	}

	fetchedFeed, entries, err := utils.PullFeed(feed.Subscription, feed.Etag)
	if err != nil {
		log.Error(err)
		return
	}

	fetchedFeed.ID = feed.ID

	if err = s.feedsRepo.Update(userID, &fetchedFeed); err != nil {
		log.Error(err)
		return
	}

	for idx := range entries {
		if _, found := s.entriesRepo.EntryWithGUID(userID, entries[idx].GUID); !found {
			entries[idx].ID = utils.CreateID()
			entries[idx].Feed = *feed
			s.entriesRepo.Create(userID, &entries[idx])
		}
	}
}

func (s *Service) SyncUser(userID string) {
	var (
		feeds          []models.Feed
		continuationID string
	)

	for {
		feeds, continuationID = s.feedsRepo.List(userID, models.Page{
			ContinuationID: continuationID,
			Count:          maxPageSizePerThread,
		})

		for idx := range feeds {
			s.updateFeed(userID, &feeds[idx])
		}

		if continuationID == "" {
			break
		}
	}
}

func (s *Service) syncUsers() {
	for {
		// List up to maxThreads of users per iteration
		users, continuationID := s.usersRepo.List(models.Page{ContinuationID: "", Count: maxThreads})
		if len(users) == 0 {
			break
		}

		for idx := range users {
			s.userQueue <- users[idx]
		}

		if continuationID == "" {
			break
		}
	}
}

func (s *Service) scheduleTask() {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.syncUsers()
			case <-s.quit:
				return
			}
		}
	}()
}

// Start a SyncService
func (s *Service) Start(interval time.Duration) {
	s.interval = interval

	s.userQueue = make(chan models.User, maxThreads)

	s.wg.Add(maxThreads)

	for i := 0; i < maxThreads; i++ {
		go s.syncUserHandler()
	}

	s.ticker = time.NewTicker(s.interval)
	s.scheduleTask()
}

// Stop a SyncService
func (s *Service) Stop() {
	s.ticker.Stop()

	s.quit <- true

	close(s.userQueue)

	s.wg.Wait()
}

// NewService creates a new SyncService object
func NewService(feedsRepo repo.Feeds, usersRepo repo.Users, entriesRepo repo.Entries) Service {
	return Service{
		quit:        make(chan bool),
		wg:          sync.WaitGroup{},
		feedsRepo:   feedsRepo,
		usersRepo:   usersRepo,
		entriesRepo: entriesRepo,
	}
}
