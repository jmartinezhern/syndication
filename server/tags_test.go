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

func (s *ServerTestSuite) TestNewTag() {
	payload := []byte(`{"name": "` + RandStringRunes(8) + `"}`)
	req, err := http.NewRequest("POST", testBaseURL+"/v1/tags", bytes.NewBuffer(payload))
	s.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusCreated, resp.StatusCode)

	respTag := new(models.Tag)
	err = json.NewDecoder(resp.Body).Decode(respTag)
	s.Require().Nil(err)

	s.Require().NotEmpty(respTag.APIID)
	s.NotEmpty(respTag.Name)

	dbTag, found := s.db.TagWithAPIID(respTag.APIID)
	s.True(found)
	s.Equal(dbTag.Name, respTag.Name)
}

func (s *ServerTestSuite) TestConflictingNewTag() {
	tagName := "Sports"
	s.db.NewTag(tagName)

	payload := []byte(`{"name": "` + tagName + `"}`)
	req, err := http.NewRequest("POST", testBaseURL+"/v1/tags", bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusConflict, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetTags() {
	for i := 0; i < 5; i++ {
		tag := s.db.NewTag("Tag " + strconv.Itoa(i+1))
		s.Require().NotEmpty(tag.APIID)
	}

	req, err := http.NewRequest("GET", testBaseURL+"/v1/tags", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	respTags := new(Tags)
	err = json.NewDecoder(resp.Body).Decode(respTags)
	s.Require().Nil(err)

	s.Len(respTags.Tags, 5)
}

func (s *ServerTestSuite) TestGetTag() {
	tag := s.db.NewTag("News")
	s.Require().NotEmpty(tag.APIID)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/tags/"+tag.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	respTag := new(models.Tag)
	err = json.NewDecoder(resp.Body).Decode(respTag)
	s.Require().Nil(err)

	s.Equal(respTag.Name, tag.Name)
	s.Equal(respTag.APIID, tag.APIID)
}

func (s *ServerTestSuite) TestGetNonExistingTag() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/tags/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestEditTag() {
	tag := s.db.NewTag("News")
	s.Require().NotEmpty(tag.APIID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/tags/"+tag.APIID, bytes.NewBuffer(payload))
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	editedTag, found := s.db.TagWithAPIID(tag.APIID)
	s.Require().True(found)
	s.Equal(editedTag.Name, "World News")
}

func (s *ServerTestSuite) TestEditNonExistingTag() {
	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/tags/123456", bytes.NewBuffer(payload))
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestDeleteTag() {
	tag := s.db.NewTag("News")
	s.Require().NotEmpty(tag.APIID)

	req, err := http.NewRequest("DELETE", testBaseURL+"/v1/tags/"+tag.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	_, found := s.db.CategoryWithAPIID(tag.APIID)
	s.False(found)
}

func (s *ServerTestSuite) TestDeleteNonExistingTag() {
	req, err := http.NewRequest("DELETE", testBaseURL+"/v1/tags/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestTagEntries() {
	tag := s.db.NewTag("News")

	feed := s.db.NewFeed("Test site", "http://example.com")

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
			Feed:   feed,
		}

		entries = append(entries, entry)
	}

	_, err := s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerAny)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	list := EntryIds{
		Entries: entryAPIIDs,
	}

	b, err := json.Marshal(list)
	s.Require().Nil(err)

	req, err := http.NewRequest("PUT", testBaseURL+"/v1/tags/"+tag.APIID+"/entries", bytes.NewBuffer(b))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	taggedEntries := s.db.EntriesFromTag(tag.APIID, models.MarkerAny, true)
	s.Len(taggedEntries, 5)
}

func (s *ServerTestSuite) TestTagEntriesWithNonExistingTag() {
	payload := []byte(`{"entries": ["12345"]}`)
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/tags/123456/entries", bytes.NewBuffer(payload))
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestGetEntriesFromTag() {
	tag := s.db.NewTag("News")
	s.Require().NotEmpty(tag.APIID)

	feed := s.db.NewFeed("World News", mockRSSServer.URL+"/rss.xml")
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerAny)
	s.Require().NotEmpty(entries)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = s.db.TagEntries(tag.APIID, entryAPIIDs)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/tags/"+tag.APIID+"/entries", nil)
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

func (s *ServerTestSuite) TestGetEntriesFromNonExistingTag() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/tags/123456/entries", nil)
	s.Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}
