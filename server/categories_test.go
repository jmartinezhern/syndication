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

func (s *ServerTestSuite) TestNewCategory() {
	payload := []byte(`{"name": "` + RandStringRunes(8) + `"}`)
	req, err := http.NewRequest("POST", testBaseURL+"/v1/categories", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusCreated, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	s.Require().Nil(err)

	s.Require().NotEmpty(respCtg.APIID)
	s.NotEmpty(respCtg.Name)

	dbCtg, found := s.db.CategoryWithAPIID(respCtg.APIID)
	s.True(found)
	s.Equal(dbCtg.Name, respCtg.Name)
}

func (s *ServerTestSuite) TestNewConflictingCategory() {
	ctgName := "Sports"
	s.db.NewCategory(ctgName)

	payload := []byte(`{"name": "` + ctgName + `"}`)
	req, err := http.NewRequest("POST", testBaseURL+"/v1/categories", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusConflict, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetCategories() {
	for i := 0; i < 5; i++ {
		ctg := s.db.NewCategory("Category " + strconv.Itoa(i+1))
		s.Require().NotEmpty(ctg.APIID)
	}

	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	type Categories struct {
		Categories []models.Category `json:"categories"`
	}

	respCtgs := new(Categories)
	err = json.NewDecoder(resp.Body).Decode(respCtgs)
	s.Require().Nil(err)

	s.Len(respCtgs.Categories, 6)
}

func (s *ServerTestSuite) TestGetCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotEmpty(ctg.APIID)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/"+ctg.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	s.Require().Nil(err)

	s.Equal(respCtg.Name, ctg.Name)
	s.Equal(respCtg.APIID, ctg.APIID)
}

func (s *ServerTestSuite) TestGetNonExistingCategory() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestEditCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotEmpty(ctg.APIID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/categories/"+ctg.APIID, bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	editedCtg, found := s.db.CategoryWithAPIID(ctg.APIID)
	s.Require().True(found)
	s.Equal(editedCtg.Name, "World News")
}

func (s *ServerTestSuite) TestEditNonExistingCategory() {
	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/categories/123456", bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestDeleteCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotEmpty(ctg.APIID)

	req, err := http.NewRequest("DELETE", testBaseURL+"/v1/categories/"+ctg.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	_, found := s.db.CategoryWithAPIID(ctg.APIID)
	s.False(found)
}

func (s *ServerTestSuite) TestDeleteNonExistingCategory() {
	req, err := http.NewRequest("DELETE", testBaseURL+"/v1/categories/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetFeedsFromCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotEmpty(ctg.APIID)

	_, err := s.db.NewFeedWithCategory("Test feed", testBaseURL, ctg.APIID)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/"+ctg.APIID+"/feeds", nil)
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
	s.Len(respFeeds.Feeds, 1)
}

func (s *ServerTestSuite) TestGetFeedsFromNonExistingCategory() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/123456/feeds", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetEntriesFromCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("World News", s.ts.URL, ctg.APIID)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/"+ctg.APIID+"/entries", nil)
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

func (s *ServerTestSuite) TestGetEntriesFromNonExistentCategory() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/123456/entries", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestMarkCategory() {
	ctg := s.db.NewCategory("News")
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("World News", s.ts.URL, ctg.APIID)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.MarkerUnread)
	s.Require().Len(entries, 5)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.MarkerRead)
	s.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", testBaseURL+"/v1/categories/"+ctg.APIID+"/mark?as=read", nil)

	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.MarkerUnread)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromCategory(ctg.APIID, true, models.MarkerRead)
	s.Require().Len(entries, 5)
}

func (s *ServerTestSuite) TestMarkNonExistingCategory() {
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/categories/123456/mark?as=read", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestAddFeedsToCategory() {
	feed := s.db.NewFeed("Example Feed", "http://example.com/feed")

	ctg := s.db.NewCategory("Test")

	type FeedList struct {
		Feeds []string `json:"feeds"`
	}

	list := FeedList{
		Feeds: []string{feed.APIID},
	}

	b, err := json.Marshal(list)
	s.Require().Nil(err)

	req, err := http.NewRequest("PUT", testBaseURL+"/v1/categories/"+ctg.APIID+"/feeds", bytes.NewBuffer(b))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	feed, found := s.db.FeedWithAPIID(feed.APIID)
	s.True(found)
}

func (s *ServerTestSuite) TestAddFeedsToNonExistingCategory() {
	payload := []byte(`{"feeds": ["123456"]}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/categories/123456/feeds", bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)

}

func (s *ServerTestSuite) TestGetStatsForCategory() {
	ctg := s.db.NewCategory("World")
	s.Require().NotEmpty(ctg.APIID)

	feed, err := s.db.NewFeedWithCategory("News", "http://example.com", ctg.APIID)
	s.Require().Nil(err)
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

	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/"+ctg.APIID+"/stats", nil)
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

func (s *ServerTestSuite) TestGetStatsForNonExistentCategory() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/categories/123456/stats", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}
