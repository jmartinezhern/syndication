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
	"net/http"
	"time"

	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/models"

	"github.com/jasonlvhit/gocron"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
)

// Sync represents a syncing worker.
type Sync struct {
	scheduler   *gocron.Scheduler
	cronChannel chan bool
	db          *database.DB
}

func (s *Sync) checkForUpdates(feed *models.Feed, user *models.User) ([]models.Entry, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", feed.Subscription, nil)
	if err != nil {
		return nil, err
	}

	if feed.Etag != "" {
		req.Header.Add("If-None-Match", feed.Etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.ContentLength <= 0 {
		return nil, nil
	}

	fp := gofeed.NewParser()
	fetchedFeed, err := fp.Parse(resp.Body)

	if err != nil {
		return nil, err
	}

	if fetchedFeed == nil {
		return nil, nil
	}

	if fetchedFeed.UpdatedParsed != nil {
		if !fetchedFeed.UpdatedParsed.After(feed.LastUpdated) {
			return nil, nil
		}
	}

	if fetchedFeed.Items == nil || len(fetchedFeed.Items) == 0 {
		return nil, nil
	}

	var entries []models.Entry
	for _, item := range fetchedFeed.Items {
		var itemGUID string
		if item.GUID != "" {
			itemGUID = item.GUID
		} else {
			itemHash := md5.Sum([]byte(item.Title + item.Link))
			itemGUID = string(itemHash[:md5.Size])
			item.GUID = itemGUID
		}

		if s.db.EntryWithGUIDExists(itemGUID, user) {
			continue
		}

		entries = append(entries, convertItemsToEntries(*feed, item))
	}

	feed.Title = fetchedFeed.Title
	feed.Description = fetchedFeed.Description
	feed.Source = fetchedFeed.Link
	feed.LastUpdated = time.Now()

	err = resp.Body.Close()
	if err != nil {
		log.Error(err)
	}

	return entries, nil
}

func convertItemsToEntries(feed models.Feed, item *gofeed.Item) models.Entry {
	entry := models.Entry{
		Title:       item.Title,
		Description: item.Description,
		Link:        item.Link,
		GUID:        item.GUID,
		Mark:        models.Unread,

		Feed:   feed,
		FeedID: feed.ID,
	}

	if item.Author != nil {
		entry.Author = item.Author.Name
	}

	return entry
}

// FetchFeed fetches a feed and populates a Feed model.
func FetchFeed(feed *models.Feed) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", feed.Subscription, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	fp := gofeed.NewParser()
	fetchedFeed, err := fp.Parse(resp.Body)
	if err != nil {
		return err
	}

	if feed.Title == "" {
		feed.Title = fetchedFeed.Title
	}

	feed.Description = fetchedFeed.Description
	feed.Source = fetchedFeed.Link

	err = resp.Body.Close()
	if err != nil {
		log.Error(err)
		// Only report this error
		err = nil
	}

	return err
}

// SyncUsers sync's all user's feeds.
func (s *Sync) SyncUsers() {
	users := s.db.Users()
	for _, user := range users {
		if err := s.SyncUser(&user); err != nil {
			log.Error(err)
		}
	}
}

// SyncFeed owned by user
func (s *Sync) SyncFeed(feed *models.Feed, user *models.User) error {
	if !time.Now().After(feed.LastUpdated.Add(time.Minute)) {
		return nil
	}

	entries, err := s.checkForUpdates(feed, user)
	if err != nil {
		return err
	}

	err = s.db.NewEntries(entries, *feed, user)
	if err != nil {
		return err
	}

	err = s.db.EditFeed(feed, user)
	if err != nil {
		return err
	}
	return nil
}

// SyncCategory owned by user.
func (s *Sync) SyncCategory(category *models.Category, user *models.User) error {
	feeds, err := s.db.FeedsFromCategory(category.UUID, user)
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		feed.Category = *category
		feed.CategoryID = category.ID
		err = s.SyncFeed(&feed, user)
		if err != nil {
			log.Error(err)
		}
	}

	return nil
}

// SyncUser sync's all feeds owned by user
func (s *Sync) SyncUser(user *models.User) error {
	feeds := s.db.Feeds(user)
	for _, feed := range feeds {
		if err := s.SyncFeed(&feed, user); err != nil {
			log.Error(err)
		}
	}

	return nil
}

// Start a syncer
func (s *Sync) Start() {
	s.scheduler.Every(5).Minutes().Do(s.SyncUsers)
	s.scheduler.RunAll()
	s.cronChannel = s.scheduler.Start()
}

// Stop a syncer
func (s *Sync) Stop() {
	s.cronChannel <- true
}

// NewSync creates a new Sync object
func NewSync(db *database.DB) *Sync {
	return &Sync{
		db:        db,
		scheduler: gocron.NewScheduler(),
	}
}
