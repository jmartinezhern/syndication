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
	"net/http/httptest"

	"github.com/labstack/echo"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

func (t *ServerTestSuite) TestGetEntry() {
	feed, err := t.server.feeds.New("example", "http://localhost:9090", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
	}, feed.APIID, t.user)

	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetParamNames("entryID")
	c.SetParamValues(entry.APIID)
	c.SetPath("/v1/entries/:entryID")

	t.NoError(t.server.GetEntry(c))
	t.Equal(http.StatusOK, t.rec.Code)

	var sEntry models.Entry
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &sEntry))

	t.Equal(entry.Title, sEntry.Title)
}

func (t *ServerTestSuite) TestGetUnknownEntry() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetParamNames("entryID")
	c.SetParamValues("bogus")
	c.SetPath("/v1/entries/:entryID")

	t.EqualError(
		t.server.GetEntry(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestGetEntries() {
	feed, err := t.server.feeds.New("example", "http://localhost:9090", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/entries")

	t.NoError(t.server.GetEntries(c))

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &entries))

	t.Len(entries.Entries, 1)
	t.Equal(entry.Title, entries.Entries[0].Title)
}

func (t *ServerTestSuite) TestMarkEntry() {
	feed, err := t.server.feeds.New("example", "http://localhost:9090", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	entries, _ := database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Empty(entries)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetParamNames("entryID")
	c.SetParamValues(entry.APIID)
	c.SetPath("/v1/entries/:entryID/mark")

	t.NoError(t.server.MarkEntry(c))
}

func (t *ServerTestSuite) TestMarkUnknownEntry() {
	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetParamNames("entryID")
	c.SetParamValues("bogus")
	c.SetPath("/v1/entries/:entryID/mark")

	t.EqualError(
		t.server.MarkEntry(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestMarkAllEntries() {
	feed, err := t.server.feeds.New("example", "http://localhost:9090", t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	_, err = database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	entries, _ := database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Empty(entries)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/entries/mark")

	t.NoError(t.server.MarkAllEntries(c))
}

func (t *ServerTestSuite) TestGetEntryStats() {
	req := httptest.NewRequest(echo.PUT, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/entries/stats")

	t.NoError(t.server.GetEntryStats(c))

	var stats models.Stats
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), &stats))
}
