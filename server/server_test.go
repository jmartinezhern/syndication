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

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/plugins"
	"github.com/varddum/syndication/sync"
)

const TestDBPath = "/tmp/syndication-test-server.db"

type (
	ServerTestSuite struct {
		suite.Suite

		db     *database.DB
		sync   *sync.Sync
		server *Server
		user   models.User
		token  string
		ts     *httptest.Server
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (s *ServerTestSuite) SetupTest() {
	randUserName := RandStringRunes(8)

	user := s.db.NewUser(randUserName, "testtesttest")
	s.Require().NotEmpty(user.APIID)

	token, err := s.db.NewAPIKey(s.server.config.AuthSecret, &user)
	s.Require().Nil(err)
	s.Require().NotEmpty(token.Key)

	s.token = token.Key
	s.user = user
}

func (s *ServerTestSuite) TearDownTest() {
	s.db.DeleteUser(s.user.APIID)
}

func (s *ServerTestSuite) TestPlugins() {
	req, err := http.NewRequest("GET", "http://localhost:9876/api_test/hello_world", nil)
	s.Require().Nil(err)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)
}

func (s *ServerTestSuite) TestNewFeed() {
	payload := []byte(`{"title":"RSS Test", "subscription": "` + s.ts.URL + `"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/feeds", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(201, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	s.Require().Nil(err)

	s.Require().NotEmpty(respFeed.APIID)
	s.NotEmpty(respFeed.Title)

	dbFeed, found := s.db.FeedWithAPIID(respFeed.APIID, &s.user)
	s.Require().True(found)
	s.Equal(dbFeed.Title, respFeed.Title)
}

func (s *ServerTestSuite) TestNewUnretrivableFeed() {
	payload := []byte(`{"title":"EFF", "subscription": "https://localhost:17170/rss/updates.xml"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/feeds", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(400, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetFeeds() {
	for i := 0; i < 5; i++ {
		feed := s.db.NewFeed("Feed "+strconv.Itoa(i+1), "http://example.com/feed", &s.user)
		s.Require().NotEmpty(feed.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	s.Require().Nil(err)
	s.Len(respFeeds.Feeds, 5)
}

func (s *ServerTestSuite) TestGetFeed() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds/"+feed.APIID, nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	s.Require().Nil(err)

	s.Equal(feed.Title, respFeed.Title)
	s.Equal(feed.APIID, respFeed.APIID)
}

func (s *ServerTestSuite) TestEditFeed() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	payload := []byte(`{"title": "EFF Updates"}`)
	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/feeds/"+feed.APIID, bytes.NewBuffer(payload))
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	respFeed, found := s.db.FeedWithAPIID(feed.APIID, &s.user)
	s.True(found)
	s.Equal(respFeed.Title, "EFF Updates")
}

func (s *ServerTestSuite) TestDeleteFeed() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	req, err := http.NewRequest("DELETE", "http://localhost:9876/v1/feeds/"+feed.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	_, found := s.db.FeedWithAPIID(feed.APIID, &s.user)
	s.False(found)
}

func (s *ServerTestSuite) TestGetEntriesFromFeed() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.db.NewFeed(feed.Title, feed.Subscription, &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.server.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds/"+feed.APIID+"/entries", nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	s.Require().Nil(err)
	s.Len(respEntries.Entries, 5)
}

func (s *ServerTestSuite) TestMarkFeed() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.server.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Unread, &s.user)
	s.Require().Len(entries, 5)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Read, &s.user)
	s.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/feeds/"+feed.APIID+"/mark?as=read", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Unread, &s.user)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Read, &s.user)
	s.Require().Len(entries, 5)
}

func (s *ServerTestSuite) TestNewCategory() {
	payload := []byte(`{"name": "` + RandStringRunes(8) + `"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/categories", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(201, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	s.Require().Nil(err)

	s.Require().NotEmpty(respCtg.APIID)
	s.NotEmpty(respCtg.Name)

	dbCtg, found := s.db.CategoryWithAPIID(respCtg.APIID, &s.user)
	s.True(found)
	s.Equal(dbCtg.Name, respCtg.Name)
}

func (s *ServerTestSuite) TestNewTag() {
	payload := []byte(`{"name": "` + RandStringRunes(8) + `"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/tags", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(201, resp.StatusCode)

	respTag := new(models.Tag)
	err = json.NewDecoder(resp.Body).Decode(respTag)
	s.Require().Nil(err)

	s.Require().NotEmpty(respTag.APIID)
	s.NotEmpty(respTag.Name)

	dbTag, found := s.db.TagWithAPIID(respTag.APIID, &s.user)
	s.True(found)
	s.Equal(dbTag.Name, respTag.Name)
}

func (s *ServerTestSuite) TestGetCategories() {
	for i := 0; i < 5; i++ {
		ctg := s.db.NewCategory("Category "+strconv.Itoa(i+1), &s.user)
		s.Require().NotEmpty(ctg.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Categories struct {
		Categories []models.Category `json:"categories"`
	}

	respCtgs := new(Categories)
	err = json.NewDecoder(resp.Body).Decode(respCtgs)
	s.Require().Nil(err)

	s.Len(respCtgs.Categories, 6)
}

func (s *ServerTestSuite) TestGetTags() {
	for i := 0; i < 5; i++ {
		tag := s.db.NewTag("Tag "+strconv.Itoa(i+1), &s.user)
		s.Require().NotEmpty(tag.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/tags", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	respTags := new(Tags)
	err = json.NewDecoder(resp.Body).Decode(respTags)
	s.Require().Nil(err)

	s.Len(respTags.Tags, 5)
}

func (s *ServerTestSuite) TestGetCategory() {
	ctg := s.db.NewCategory("News", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+ctg.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	s.Require().Nil(err)

	s.Equal(respCtg.Name, ctg.Name)
	s.Equal(respCtg.APIID, ctg.APIID)
}

func (s *ServerTestSuite) TestGetTag() {
	tag := s.db.NewTag("News", &s.user)
	s.Require().NotEmpty(tag.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/tags/"+tag.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respTag := new(models.Tag)
	err = json.NewDecoder(resp.Body).Decode(respTag)
	s.Require().Nil(err)

	s.Equal(respTag.Name, tag.Name)
	s.Equal(respTag.APIID, tag.APIID)
}

func (s *ServerTestSuite) TestEditCategory() {
	ctg := s.db.NewCategory("News", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/categories/"+ctg.APIID, bytes.NewBuffer(payload))
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	editedCtg, found := s.db.CategoryWithAPIID(ctg.APIID, &s.user)
	s.Require().True(found)
	s.Equal(editedCtg.Name, "World News")
}

func (s *ServerTestSuite) TestEditTag() {
	tag := s.db.NewTag("News", &s.user)
	s.Require().NotEmpty(tag.APIID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/tags/"+tag.APIID, bytes.NewBuffer(payload))
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	editedTag, found := s.db.TagWithAPIID(tag.APIID, &s.user)
	s.Require().True(found)
	s.Equal(editedTag.Name, "World News")
}

func (s *ServerTestSuite) TestDeleteCategory() {
	ctg := s.db.NewCategory("News", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	req, err := http.NewRequest("DELETE", "http://localhost:9876/v1/categories/"+ctg.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	_, found := s.db.CategoryWithAPIID(ctg.APIID, &s.user)
	s.False(found)
}

func (s *ServerTestSuite) TestDeleteTag() {
	tag := s.db.NewTag("News", &s.user)
	s.Require().NotEmpty(tag.APIID)

	req, err := http.NewRequest("DELETE", "http://localhost:9876/v1/tags/"+tag.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	_, found := s.db.CategoryWithAPIID(tag.APIID, &s.user)
	s.False(found)
}

func (s *ServerTestSuite) TestGetFeedsFromCategory() {
	ctg := s.db.NewCategory("News", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	_, err := s.db.NewFeedWithCategory("Test feed", "http://localhost:9876", ctg.APIID, &s.user)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+ctg.APIID+"/feeds", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	s.Require().Nil(err)
	s.Len(respFeeds.Feeds, 1)
}

func (s *ServerTestSuite) TestGetEntriesFromCategory() {
	ctg := s.db.NewCategory("News", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("World News", s.ts.URL, ctg.APIID, &s.user)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	err = s.server.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+ctg.APIID+"/entries", nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	s.Require().Nil(err)
	s.Len(respEntries.Entries, 5)
}

func (s *ServerTestSuite) TestGetEntriesFromTag() {
	tag := s.db.NewTag("News", &s.user)
	s.Require().NotEmpty(tag.APIID)

	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.server.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)
	s.Require().NotEmpty(entries)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = s.db.TagEntries(tag.APIID, entryAPIIDs, &s.user)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/tags/"+tag.APIID+"/entries", nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	s.Require().Nil(err)
	s.Len(respEntries.Entries, 5)
}

func (s *ServerTestSuite) TestTagEntries() {
	tag := s.db.NewTag("News", &s.user)

	feed := s.db.NewFeed("Test site", "http://example.com", &s.user)

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.Unread,
			Feed:   feed,
		}

		entries = append(entries, entry)
	}

	_, err := s.db.NewEntries(entries, feed.APIID, &s.user)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Any, &s.user)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	list := EntryIds{
		Entries: entryAPIIDs,
	}

	b, err := json.Marshal(list)
	s.Require().Nil(err)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/tags/"+tag.APIID+"/entries", bytes.NewBuffer(b))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	taggedEntries := s.db.EntriesFromTag(tag.APIID, models.Any, true, &s.user)
	s.Len(taggedEntries, 5)
}

func (s *ServerTestSuite) TestMarkCategory() {
	ctg := s.db.NewCategory("News", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("World News", s.ts.URL, ctg.APIID, &s.user)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	s.server.sync.SyncFeed(&feed, &s.user)

	entries := s.db.EntriesFromCategory(ctg.APIID, true, models.Unread, &s.user)
	s.Require().Len(entries, 5)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.Read, &s.user)
	s.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/categories/"+ctg.APIID+"/mark?as=read", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.Unread, &s.user)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.Read, &s.user)
	s.Require().Len(entries, 5)
}

func (s *ServerTestSuite) TestGetEntries() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.server.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries", nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	s.Require().Nil(err)
	s.Len(respEntries.Entries, 5)
}

func (s *ServerTestSuite) TestGetEntry() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	entry := models.Entry{
		Title: "Item 1",
		Link:  "http://localhost:9876/item_1",
	}

	entry, err := s.db.NewEntry(entry, feed.APIID, &s.user)
	s.Require().Nil(err)
	s.Require().NotEmpty(entry.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries/"+entry.APIID, nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respEntry := new(models.Entry)
	err = json.NewDecoder(resp.Body).Decode(respEntry)
	s.Require().Nil(err)

	s.Equal(entry.Title, respEntry.Title)
	s.Equal(entry.APIID, respEntry.APIID)
}

func (s *ServerTestSuite) TestMarkEntry() {
	feed := s.db.NewFeed("World News", s.ts.URL, &s.user)
	s.Require().NotEmpty(feed.APIID)

	err := s.server.sync.SyncFeed(&feed, &s.user)
	s.Require().Nil(err)

	entries := s.db.EntriesFromFeed(feed.APIID, true, models.Read, &s.user)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Unread, &s.user)
	s.Require().Len(entries, 5)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/entries/"+entries[0].APIID+"/mark?as=read", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(204, resp.StatusCode)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Unread, &s.user)
	s.Require().Len(entries, 4)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.Read, &s.user)
	s.Require().Len(entries, 1)
}

func (s *ServerTestSuite) TestGetStatsForFeed() {
	feed := s.db.NewFeed("News", "http://example.com", &s.user)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.Read,
			Saved: true,
		}

		s.db.NewEntry(entry, feed.APIID, &s.user)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.Unread,
		}

		s.db.NewEntry(entry, feed.APIID, &s.user)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds/"+feed.APIID+"/stats", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	s.Require().Nil(err)

	s.Equal(7, respStats.Unread)
	s.Equal(3, respStats.Read)
	s.Equal(3, respStats.Saved)
	s.Equal(10, respStats.Total)
}

func (s *ServerTestSuite) TestGetStats() {
	feed := s.db.NewFeed("News", "http://example.com", &s.user)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.Read,
			Saved: true,
		}

		s.db.NewEntry(entry, feed.APIID, &s.user)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.Unread,
		}

		s.db.NewEntry(entry, feed.APIID, &s.user)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries/stats", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	s.Require().Nil(err)

	s.Equal(7, respStats.Unread)
	s.Equal(3, respStats.Read)
	s.Equal(3, respStats.Saved)
	s.Equal(10, respStats.Total)
}

func (s *ServerTestSuite) TestGetStatsForCategory() {
	ctg := s.db.NewCategory("World", &s.user)
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("News", "http://example.com", ctg.APIID, &s.user)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.Read,
			Saved: true,
		}

		s.db.NewEntry(entry, feed.APIID, &s.user)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.Unread,
		}

		s.db.NewEntry(entry, feed.APIID, &s.user)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+ctg.APIID+"/stats", nil)

	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	s.Require().Nil(err)

	s.Equal(7, respStats.Unread)
	s.Equal(3, respStats.Read)
	s.Equal(3, respStats.Saved)
	s.Equal(10, respStats.Total)
}

func (s *ServerTestSuite) TestAddFeedsToCategory() {
	feed := s.db.NewFeed("Example Feed", "http://example.com/feed", &s.user)

	ctg := s.db.NewCategory("Test", &s.user)

	type FeedList struct {
		Feeds []string `json:"feeds"`
	}

	list := FeedList{
		Feeds: []string{feed.APIID},
	}

	b, err := json.Marshal(list)
	s.Require().Nil(err)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/categories/"+ctg.APIID+"/feeds", bytes.NewBuffer(b))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	feed, found := s.db.FeedWithAPIID(feed.APIID, &s.user)
	s.True(found)
}

func (s *ServerTestSuite) TestRegister() {
	s.db.DeleteUser(s.user.APIID)

	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(204, regResp.StatusCode)

	users := s.db.Users("username")
	s.Len(users, 1)

	s.Equal(randUserName, users[0].Username)
	s.NotEmpty(users[0].APIID)

	err = regResp.Body.Close()
	s.Nil(err)

	s.db.DeleteUser(users[0].APIID)
}

func (s *ServerTestSuite) TestLogin() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(204, regResp.StatusCode)

	err = regResp.Body.Close()
	s.Require().Nil(err)

	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(200, loginResp.StatusCode)

	type Token struct {
		Token string `json:"token"`
	}

	var token Token
	err = json.NewDecoder(loginResp.Body).Decode(&token)
	s.Require().Nil(err)
	s.NotEmpty(token.Token)

	user, found := s.db.UserWithName(randUserName)
	s.True(found)

	err = loginResp.Body.Close()
	s.Nil(err)

	s.db.DeleteUser(user.APIID)
}

func (s *ServerTestSuite) TestLoginWithNonExistentUser() {
	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {"bogus"}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(401, loginResp.StatusCode)

	err = loginResp.Body.Close()
	s.Nil(err)
}

func (s *ServerTestSuite) TestLoginWithBadPassword() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	user, found := s.db.UserWithName(randUserName)
	s.Require().True(found)
	defer s.db.DeleteUser(user.APIID)

	s.Equal(204, regResp.StatusCode)

	err = regResp.Body.Close()
	s.Require().Nil(err)

	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"bogus"}})
	s.Require().Nil(err)

	s.Equal(401, loginResp.StatusCode)

	err = loginResp.Body.Close()
	s.Nil(err)
}

func (s *ServerTestSuite) startServer() {
	conf := config.DefaultConfig
	conf.Server.HTTPPort = 9876
	conf.Server.AuthSecret = "secret"
	conf.Server.EnableRequestLogs = false

	var err error
	s.db, err = database.NewDB(config.Database{
		Type:       "sqlite3",
		Connection: TestDBPath,
		APIKeyExpiration: config.Duration{
			Duration: time.Hour * 72,
		},
	})
	s.Require().Nil(err)

	s.sync = sync.NewSync(s.db, config.Sync{
		SyncInterval: config.Duration{Duration: time.Second * 5},
	})

	if s.server == nil {
		plgnPath := []string{os.Getenv("GOPATH") + "/src/github.com/varddum/syndication/api.so"}
		plgns := plugins.NewPlugins(plgnPath)

		s.server = NewServer(s.db, s.sync, &plgns, conf.Server)
		s.server.handle.HideBanner = true
		go s.server.Start()
	}

	time.Sleep(time.Second)

	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:9876</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:9876/item_1</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:9876/item_2</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:9876/item_3</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:9876/item_4</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:9876/item_5</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	s.ts = httptest.NewServer(http.HandlerFunc(handler))
}

func TestServerTestSuite(t *testing.T) {
	serverSuite := new(ServerTestSuite)
	serverSuite.startServer()
	suite.Run(t, serverSuite)
	serverSuite.server.Stop()
	os.Remove(TestDBPath)
	serverSuite.ts.Close()
}
