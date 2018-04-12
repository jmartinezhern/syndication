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
	"encoding/json"
	"net/http"

	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/sync"
)

func (s *ServerTestSuite) TestGetEntries() {
	feed := s.db.NewFeed("World News", s.ts.URL)
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/entries", nil)
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

func (s *ServerTestSuite) TestGetEntry() {
	feed := s.db.NewFeed("World News", s.ts.URL)
	s.Require().NotEmpty(feed.APIID)

	entry := models.Entry{
		Title: "Item 1",
		Link:  testBaseURL + "/item_1",
	}

	entry, err := s.db.NewEntry(entry, feed.APIID)
	s.Require().Nil(err)
	s.Require().NotEmpty(entry.APIID)

	req, err := http.NewRequest("GET", testBaseURL+"/v1/entries/"+entry.APIID, nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	respEntry := new(models.Entry)
	err = json.NewDecoder(resp.Body).Decode(respEntry)
	s.Require().Nil(err)

	s.Equal(entry.Title, respEntry.Title)
	s.Equal(entry.APIID, respEntry.APIID)
}

func (s *ServerTestSuite) TestGetNonExistentEntry() {
	req, err := http.NewRequest("GET", testBaseURL+"/v1/entries/123456", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestMarkEntry() {
	feed := s.db.NewFeed("World News", s.ts.URL)
	s.Require().NotEmpty(feed.APIID)

	entries, err := sync.PullFeed(&feed)
	s.Require().Nil(err)

	_, err = s.db.NewEntries(entries, feed.APIID)
	s.Require().Nil(err)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerRead)
	s.Require().Len(entries, 0)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerUnread)
	s.Require().Len(entries, 5)

	req, err := http.NewRequest("PUT", testBaseURL+"/v1/entries/"+entries[0].APIID+"/mark?as=read", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerUnread)
	s.Require().Len(entries, 4)

	entries = s.db.EntriesFromFeed(feed.APIID, true, models.MarkerRead)
	s.Require().Len(entries, 1)
}

func (s *ServerTestSuite) TestMarkNonExistentEntry() {
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/entries/123456/mark?as=read", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ServerTestSuite) TestMarkEntryWithoutMarker() {
	req, err := http.NewRequest("PUT", testBaseURL+"/v1/entries/123456/mark", nil)
	s.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	s.Require().Nil(err)
	defer resp.Body.Close()

	s.Equal(http.StatusBadRequest, resp.StatusCode)
}
