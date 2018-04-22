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

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"

	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
)

const maxThreads = 100

var (
	// ErrParsingFeed Signals that a an error occurred while processing
	// a RSS or Atom Feed
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
	ticker        *time.Ticker
	db            *database.DB
	userPool      userPool
	userWaitGroup sync.WaitGroup
	status        chan syncStatus
	interval      time.Duration
	dbLock        sync.Mutex
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
	users := s.db.Users()
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
func PullFeed(feed *models.Feed) ([]models.Entry, error) {
	fetchedFeed, err := fetchFeed(feed.Subscription, feed.Etag)
	if err != nil {
		return nil, err
	}

	feed.Title = fetchedFeed.Title
	feed.Description = fetchedFeed.Description
	feed.Source = fetchedFeed.Link
	feed.LastUpdated = time.Now()

	entries := make([]models.Entry, len(fetchedFeed.Items))
	for idx, item := range fetchedFeed.Items {
		entries[idx] = convertItemToEntry(item)
	}

	return entries, nil
}

// SyncUser sync's all feeds owned by user
func (s *Service) SyncUser(user *models.User) error {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	userDB := s.db.NewUserDB(*user)
	feeds := userDB.Feeds()
	for _, feed := range feeds {
		if !time.Now().After(feed.LastUpdated.Add(s.interval)) {
			continue
		}

		fetchedEntries, err := PullFeed(&feed)
		if err != nil {
			log.Error(err)
		}

		err = userDB.EditFeed(&feed)
		if err != nil {
			log.Error(err)
		}

		entries := []models.Entry{}
		for _, entry := range fetchedEntries {
			if found := userDB.EntryWithGUIDExists(entry.GUID, feed.APIID); !found {
				entries = append(entries, entry)
			}
		}

		_, err = userDB.NewEntries(entries, feed.APIID)
		if err != nil {
			log.Error(err)
		}
	}

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
func NewService(db *database.DB, syncInterval time.Duration) Service {
	return Service{
		db:       db,
		status:   make(chan syncStatus),
		interval: syncInterval,
	}
}
