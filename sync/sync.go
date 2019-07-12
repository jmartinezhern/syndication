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
	userQueue       chan models.User
	dbLock          sync.Mutex

	interval        time.Duration

	usersRepo       repo.Users
	feedsRepo       repo.Feeds
	entriesRepo     repo.Entries
}

// SyncUsers sync's all user's feeds.
func (s *Service) SyncUsers() {
	s.userQueue = make(chan models.User)

	// List up to maxThreads of users per iteration
	users, continuationID := s.usersRepo.List("", maxThreads)
	if len(users) == 0 {
		return
	}

	s.userWaitGroup.Add(len(users))

	// We may have less users than we do maxThreads.
	// Start length of users of goroutines which cannot be more than maxThreads.
	for i := 0; i < len(users); i++ {
		go func() {
			for {
				user, more := <-s.userQueue
				if !more {
					return
				}
				s.SyncUser(user.ID)
				s.userWaitGroup.Done()
			}
		}()
	}

	for {
		for idx := range users {
			s.userQueue <- users[idx]
		}

		s.userWaitGroup.Wait()

		if continuationID == "" {
			break
		}

		users, continuationID = s.usersRepo.List(continuationID, maxThreads)
		if len(users) == 0 {
			break
		}

		s.userWaitGroup.Add(len(users))
	}

	close(s.userQueue)
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
	s.userWaitGroup.Wait()
	s.status <- stopping
	<-s.status
}

// NewService creates a new SyncService object
func NewService(syncInterval time.Duration, feedsRepo repo.Feeds, usersRepo repo.Users, entriesRepo repo.Entries) Service {
	return Service{
		status:          make(chan syncStatus),
		interval:        syncInterval,
		feedsRepo:       feedsRepo,
		usersRepo:       usersRepo,
		entriesRepo:     entriesRepo,
	}
}
