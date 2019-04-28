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
	maxThreads = 100
	stopping   = iota
	stopped
)

type syncStatus = int

// Service defines properties for running a Feed Sync Service.
// Service will update all feeds for all users periodically.
type Service struct {
	ticker          *time.Ticker
	userWaitGroup   sync.WaitGroup
	status          chan syncStatus
	interval        time.Duration
	deleteAfterDays int
	dbLock          sync.Mutex
	usersRepo       repo.Users
	feedsRepo       repo.Feeds
	entriesRepo     repo.Entries
}

// SyncUsers sync's all user's feeds.
func (s *Service) SyncUsers() {
	var continuationID string
	var users []models.User
	for {
		users, continuationID = s.usersRepo.List(continuationID, maxThreads)
		if len(users) == 0 {
			break
		}

		s.userWaitGroup.Add(len(users))

		for idx := range users {
			go func(user models.User) {
				s.SyncUser(user.ID)
				s.userWaitGroup.Done()
			}(users[idx])
		}

		s.userWaitGroup.Wait()

		if continuationID == "" {
			break
		}
	}
}

// SyncUser sync's all feeds owned by user
func (s *Service) SyncUser(userID string) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	continuationID := ""
	for {
		var feeds []models.Feed
		feeds, continuationID = s.feedsRepo.List(userID, continuationID, 100)
		for idx := range feeds {
			feed := feeds[idx]
			if !time.Now().After(feed.LastUpdated.Add(s.interval)) {
				continue
			}

			fetchedFeed, fetchedEntries, err := utils.PullFeed(feed.Subscription, feed.Etag)
			if err != nil {
				log.Error(err)
			}

			fetchedFeed.ID = feed.ID

			err = s.feedsRepo.Update(userID, &fetchedFeed)
			if err != nil {
				log.Error(err)
			}

			for idx := range fetchedEntries {
				entry := fetchedEntries[idx]
				if _, found := s.entriesRepo.EntryWithGUID(userID, entry.GUID); !found {
					entry.ID = utils.CreateID()
					entry.Feed = feed
					s.entriesRepo.Create(userID, &entry)
				}
			}
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
				s.SyncUsers()
			case <-s.status:
				s.ticker.Stop()
				s.status <- stopped
				return
			}
		}
	}()
}

// Start a SyncService
func (s *Service) Start() {
	s.ticker = time.NewTicker(s.interval)
	s.scheduleTask()
}

// Stop a SyncService
func (s *Service) Stop() {
	s.ticker.Stop()
	s.status <- stopping
	<-s.status
	s.userWaitGroup.Wait()
}

// NewService creates a new SyncService object
func NewService(syncInterval time.Duration, deleteAfter int, feedsRepo repo.Feeds, usersRepo repo.Users, entriesRepo repo.Entries) Service {
	return Service{
		status:          make(chan syncStatus),
		interval:        syncInterval,
		deleteAfterDays: deleteAfter,
		feedsRepo:       feedsRepo,
		usersRepo:       usersRepo,
		entriesRepo:     entriesRepo,
	}
}
