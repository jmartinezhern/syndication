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
	"net/http"
	"strconv"

	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/sync"
)

func (s *ServerTestSuite) TestNewFeed() {
	payload := []byte(`{"title":"RSS Test", "subscription": "` + mockRSSServer.URL + `/rss.xml"}`)
	req, err := http.NewRequest("POST", testBaseURL+"/v1/feeds", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusCreated, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	s.Require().Nil(err)

	s.Require().NotEmpty(respFeed.APIID)
	s.NotEmpty(respFeed.Title)

	dbFeed, found := s.db.FeedWithAPIID(respFeed.APIID)
	s.Require().True(found)
	s.Equal(dbFeed.Title, respFeed.Title)

	entries := s.db.EntriesFromFeed(respFeed.APIID, false, models.MarkerAny)
	s.Require().Len(entries, 5)

	s.Equal("Item 1", entries[0].Title)
}

func (s *ServerTestSuite) TestNewUnretrievableFeed() {
	payload := []byte(`{"title":"EFF", "subscription": "https://localhost:17170/rss/updates.xml"}`)
	req, err := http.NewRequest("POST", testBaseURL+"/v1/feeds", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetFeeds() {
	for i := 0; i < 5; i++ {
		feed := s.db.NewFeed("Feed "+strconv.Itoa(i+1), "http://example.com/feed")
		s.Require().NotEmpty(feed.APIID)
	}

	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	s.Require().Nil(err)
	s.Len(respFeeds.Feeds, 5)
}

func (s *ServerTestSuite) TestGetFeed() {
	feed := s.db.NewFeed("World News", mockRSSServer.URL+"/rss.xml")
	s.Require().NotEmpty(feed.APIID)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds/"+feed.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	s.Require().Nil(err)

	s.Equal(feed.Title, respFeed.Title)
	s.Equal(feed.APIID, respFeed.APIID)
}

func (s *ServerTestSuite) TestGetNonExistentFeed() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestEditFeed() {
	feed := s.db.NewFeed("World News", mockRSSServer.URL+"/rss.xml")
	s.Require().NotEmpty(feed.APIID)

	payload := []byte(`{"title": "EFF Updates"}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/feeds/"+feed.APIID, bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	respFeed, found := s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
	s.Equal(respFeed.Title, "EFF Updates")
}

func (s *ServerTestSuite) TestEditNonExistentFeed() {
	payload := []byte(`{"title": "EFF Updates"}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/feeds/123456", bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestDeleteFeed() {
	feed := s.db.NewFeed("World News", mockRSSServer.URL+"/rss.xml")
	s.Require().NotEmpty(feed.APIID)

	req, err := http.NewRequest("DELETE", testBaseURL+"/v1/feeds/"+feed.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	_, found := s.db.FeedWithAPIID(feed.APIID)
	s.False(found)
}

func (s *ServerTestSuite) TestDeleteNonExistentFeed() {
	req, err := http.NewRequest("DELETE", testBaseURL+"/v1/feeds/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetEntriesFromFeed() {
	feed := s.db.NewFeed("World News", mockRSSServer.URL+"/rss.xml")
	s.db.NewFeed(feed.Title, feed.Subscription)
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds/"+feed.APIID+"/entries", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	s.Require().Nil(err)
	s.Len(respEntries.Entries, 5)
}

func (s *ServerTestSuite) TestGetEntriesFromNonExistentFeed() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds/123456/entries", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestMarkFeed() {
	feed := s.db.NewFeed("World News", mockRSSServer.URL+"/rss.xml")
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerUnread)
	s.Require().Len(entries, 5)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerRead)
	s.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", testBaseURL+"/v1/feeds/"+feed.APIID+"/mark?as=read", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerUnread)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerRead)
	s.Require().Len(entries, 5)
}

func (s *ServerTestSuite) TestMarkNonExistentFeed() {
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/feeds/123456/mark?as=read", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestMarkFeedWithoutMarker() {
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/feeds/123456/mark", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetStatsForFeed() {
	feed := s.db.NewFeed("News", "http://example.com")
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.MarkerRead,
			Saved: true,
		}

		s.db.NewEntry(entry, feed.APIID)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title: "Item",
			Link:  "http://example.com",
			Mark:  models.MarkerUnread,
		}

		s.db.NewEntry(entry, feed.APIID)
	}

	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds/"+feed.APIID+"/stats", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	s.Require().Nil(err)

	s.Equal(7, respStats.Unread)
	s.Equal(3, respStats.Read)
	s.Equal(3, respStats.Saved)
	s.Equal(10, respStats.Total)
}

func (s *ServerTestSuite) TestGetStatsForNonExistentFeed() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/feeds/123456/stats", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}
