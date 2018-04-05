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
	"net/http"
	"sync"
	"time"

	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"

	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
)

const maxThreads = 100

type userPool struct {
	users []models.User
	lock  sync.Mutex
}

const (
	stopping = iota
	stopped
)

type syncStatus = int

//Error identifies error caused by database queries
type Error interface {
	String() string
	Code() int
	Error() string
}

type (
	// BadRequest is a SyncError returned when a feed could not be fetched.
	BadRequest struct {
		msg string
	}
)

func (e BadRequest) Error() string {
	return e.msg
}

func (e BadRequest) String() string {
	return "Bad Request"
}

// Code returns BadRequest's corresponding error code
func (e BadRequest) Code() int {
	return 400
}

// Sync represents a syncing worker.
type Sync struct {
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

func (s *Sync) updatedFeed(feed *models.Feed) (gofeed.Feed, bool) {
	client := &http.Client{
		CheckRedirect: (func(r *http.Request, v []*http.Request) error { return http.ErrUseLastResponse }),
	}

	req, err := http.NewRequest("GET", feed.Subscription, nil)
	if err != nil {
		log.Error(err)
		return gofeed.Feed{}, false
	}

	req.Header.Add("If-None-Match", feed.Etag)

	resp, err := client.Do(req)
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Error(err)
		}
	}()

	if err != nil {
		log.Error(err)
		return gofeed.Feed{}, false
	}

	fp := gofeed.NewParser()
	fetchedFeed, err := fp.Parse(resp.Body)

	if err != nil {
		return gofeed.Feed{}, false
	}

	if fetchedFeed == nil ||
		(fetchedFeed.UpdatedParsed != nil && !fetchedFeed.UpdatedParsed.After(feed.LastUpdated)) ||
		fetchedFeed.Items == nil || len(fetchedFeed.Items) == 0 {
		return gofeed.Feed{}, false
	}

	return *fetchedFeed, true
}

func (s *Sync) getEntriesFromUpdatedFeed(feed *models.Feed, fetchedFeed gofeed.Feed) []models.Entry {
	var entries []models.Entry
	for _, item := range fetchedFeed.Items {
		entries = append(entries, convertItemsToEntries(item))
	}

	feed.Title = fetchedFeed.Title
	feed.Description = fetchedFeed.Description
	feed.Source = fetchedFeed.Link
	feed.LastUpdated = time.Now()

	return entries
}

func convertItemsToEntries(item *gofeed.Item) models.Entry {
	entry := models.Entry{
		Title: item.Title,
		Link:  item.Link,
		GUID:  item.GUID,
		Mark:  models.Unread,
	}

	if item.Author != nil {
		entry.Author = item.Author.Name
	}

	if item.PublishedParsed != nil {
		entry.Published = *item.PublishedParsed
	} else {
		entry.Published = time.Now()
	}

	return entry
}

// SyncUsers sync's all user's feeds.
func (s *Sync) SyncUsers() {
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

// SyncFeed owned by user
func (s *Sync) SyncFeed(feed *models.Feed, user *models.User) error {
	if !time.Now().After(feed.LastUpdated.Add(time.Minute)) {
		return nil
	}

	fetchedFeed, ok := s.updatedFeed(feed)
	if !ok {
		return nil
	}

	entries := s.getEntriesFromUpdatedFeed(feed, fetchedFeed)

	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	err := s.db.EditFeed(feed, user)
	if err != nil {
		return err
	}

	_, err = s.db.NewEntries(entries, feed.APIID, user)
	return err
}

// SyncUser sync's all feeds owned by user
func (s *Sync) SyncUser(user *models.User) error {
	s.dbLock.Lock()
	feeds := s.db.Feeds(user)
	s.dbLock.Unlock()
	for _, feed := range feeds {
		if err := s.SyncFeed(&feed, user); err != nil {
			log.Error(err)
		}
	}

	return nil
}

func (s *Sync) scheduleTask() {
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

// Start a syncer
func (s *Sync) Start() {
	s.ticker = time.NewTicker(s.interval)
	s.scheduleTask()
}

// Stop a syncer
func (s *Sync) Stop() {
	s.ticker.Stop()
	s.status <- stopping
	<-s.status
	s.userWaitGroup.Wait()
}

// NewSync creates a new Sync object
func NewSync(db *database.DB, config config.Sync) *Sync {
	return &Sync{
		db:       db,
		status:   make(chan syncStatus),
		interval: config.SyncInterval.Duration,
	}
}
