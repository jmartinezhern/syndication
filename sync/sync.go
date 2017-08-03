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
	"crypto/md5"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

const maxThreads = 100

var (
	// ErrParsingFeed Signals that an error occurred while processing
	// a RSS or Atom feed
	ErrParsingFeed = errors.New("Could not parse feed")
)

type userPool struct {
	users []models.User
	lock  sync.Mutex
}

const (
	stopping = iota
	stopped
)

type syncStatus = int

// Service defines properties for running a Feed Sync Service.
// Service will update all feeds for all users periodically.
type Service struct {
	ticker          *time.Ticker
	userPool        userPool
	userWaitGroup   sync.WaitGroup
	status          chan syncStatus
	interval        time.Duration
	deleteAfterDays int
	dbLock          sync.Mutex
}

func (p *userPool) get() models.User {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.users) == 0 {
		return models.User{}
	}

	user := p.users[0]
	p.users = p.users[1:]
	return user
}

func (p *userPool) put(user models.User) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.users = append(p.users, user)
}

func fetchFeed(url, etag string) (gofeed.Feed, error) {
	client := &http.Client{
		CheckRedirect: (func(r *http.Request, v []*http.Request) error { return http.ErrUseLastResponse }),
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return gofeed.Feed{}, err
	}

	req.Header.Add("If-None-Match", etag)

	resp, err := client.Do(req)
	if err != nil {
		return gofeed.Feed{}, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Warn(err)
		}
	}()

	fetchedFeed, err := gofeed.NewParser().Parse(resp.Body)
	if err != nil {
		return gofeed.Feed{}, err
	}

	return *fetchedFeed, nil
}

func convertItemToEntry(item *gofeed.Item) models.Entry {
	entry := models.Entry{
		Title: item.Title,
		Link:  item.Link,
		Mark:  models.MarkerUnread,
	}

	if item.Author != nil {
		entry.Author = item.Author.Name
	}

	if item.PublishedParsed != nil {
		entry.Published = *item.PublishedParsed
	} else {
		entry.Published = time.Now()
	}

	if item.GUID != "" {
		entry.GUID = item.GUID
	} else {
		itemHash := md5.Sum([]byte(item.Title + item.Link))
		entry.GUID = string(itemHash[:md5.Size])
	}

	return entry
}

// SyncUsers sync's all user's feeds.
func (s *Service) SyncUsers() {
	s.dbLock.Lock()
	users := database.Users()
	s.dbLock.Unlock()

	for _, user := range users {
		s.userPool.put(user)
	}

	var numThreads int
	if len(users) > maxThreads {
		numThreads = maxThreads
	} else {
		numThreads = len(users)
	}

	for i := 0; i < numThreads; i++ {
		s.userWaitGroup.Add(1)
		go func() {
			user := s.userPool.get()
			for user.ID != 0 {
				if err := s.SyncUser(&user); err != nil {
					log.Error(err)
				}
				user = s.userPool.get()
			}

			s.userWaitGroup.Done()
		}()
	}
}

// PullFeed and return all entries for that feed. If getting the
// subscription source or parsing the response fails, this function
// will error.
func PullFeed(url, etag string) (models.Feed, []models.Entry, error) {
	fetchedFeed, err := fetchFeed(url, etag)
	if err != nil {
		return models.Feed{}, nil, err
	}

	feed := models.Feed{
		Title:       fetchedFeed.Title,
		Description: fetchedFeed.Description,
		Source:      fetchedFeed.Link,
		LastUpdated: time.Now(),
	}

	entries := make([]models.Entry, len(fetchedFeed.Items))
	for idx, item := range fetchedFeed.Items {
		entries[idx] = convertItemToEntry(item)
	}

	return feed, entries, nil
}

// SyncUser sync's all feeds owned by user
func (s *Service) SyncUser(user *models.User) error {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	continuationID := ""
	for {
		feeds, continuationID := database.Feeds(continuationID, 100, *user)
		for _, feed := range feeds {
			if !time.Now().After(feed.LastUpdated.Add(s.interval)) {
				continue
			}

			fetchedFeed, fetchedEntries, err := PullFeed(feed.Subscription, feed.Etag)
			if err != nil {
				log.Error(err)
			}

			_, err = database.EditFeed(feed.APIID, fetchedFeed, *user)
			if err != nil {
				log.Error(err)
			}

			entries := []models.Entry{}
			for _, entry := range fetchedEntries {
				if found := database.EntryWithGUIDExists(entry.GUID, feed.APIID, *user); !found {
					entries = append(entries, entry)
				}
			}

			_, err = database.NewEntries(entries, feed.APIID, *user)
			if err != nil {
				log.Error(err)
			}
		}

		if continuationID == "" {
			break
		}
	}
	database.DeleteOldEntries(time.Now().AddDate(0, 0, s.deleteAfterDays*-1), *user)

	return nil
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
func NewService(syncInterval time.Duration, deleteAfter int) Service {
	return Service{
		status:          make(chan syncStatus),
		interval:        syncInterval,
		deleteAfterDays: deleteAfter,
	}
}
