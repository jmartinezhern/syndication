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
	ticker        *time.Ticker
	userWaitGroup sync.WaitGroup
	status        chan syncStatus
	userQueue     chan models.User
	dbLock        sync.Mutex

	interval time.Duration

	usersRepo   repo.Users
	feedsRepo   repo.Feeds
	entriesRepo repo.Entries
}

// SyncUsers sync's all user's feeds.
func (s *Service) SyncUsers() {
	s.userQueue = make(chan models.User)

	// List up to maxThreads of users per iteration
	users, continuationID := s.usersRepo.List(models.Page{
		ContinuationID: "",
		Count:          maxThreads,
	})
	if len(users) == 0 {
		return
	}

	s.userWaitGroup.Add(len(users))

	// We may have less users than we do maxThreads.
	// Start length of users of goroutines which cannot be more than maxThreads.
	for range users {
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

		users, continuationID = s.usersRepo.List(models.Page{ContinuationID: continuationID, Count: maxThreads})
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

	var (
		feeds          []models.Feed
		continuationID string
	)

	for {
		feeds, continuationID = s.feedsRepo.List(userID, models.Page{ContinuationID: continuationID, Count: 100})

		for feedIdx := range feeds {
			if !time.Now().After(feeds[feedIdx].LastUpdated.Add(s.interval)) {
				continue
			}

			fetchedFeed, entries, err := utils.PullFeed(feeds[feedIdx].Subscription, feeds[feedIdx].Etag)
			if err != nil {
				log.Error(err)
				continue
			}

			fetchedFeed.ID = feeds[feedIdx].ID

			if err = s.feedsRepo.Update(userID, &fetchedFeed); err != nil {
				log.Error(err)
				continue
			}

			for idx := range entries {
				if _, found := s.entriesRepo.EntryWithGUID(userID, entries[idx].GUID); !found {
					entries[idx].ID = utils.CreateID()
					entries[idx].Feed = feeds[feedIdx]
					s.entriesRepo.Create(userID, &entries[idx])
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
func NewService(syncInterval time.Duration, feedsRepo repo.Feeds, usersRepo repo.Users,
	entriesRepo repo.Entries) Service {
	return Service{
		status:      make(chan syncStatus),
		interval:    syncInterval,
		feedsRepo:   feedsRepo,
		usersRepo:   usersRepo,
		entriesRepo: entriesRepo,
	}
}
