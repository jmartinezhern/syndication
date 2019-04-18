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

package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	EntriesControllerSuite struct {
		suite.Suite

		e           *echo.Echo
		db          *sql.DB
		feedsRepo   repo.Feeds
		entriesRepo repo.Entries
		user        *models.User
		controller  *EntriesController
	}
)

func (c *EntriesControllerSuite) TestGetEntry() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user, &feed)

	entry := models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "example",
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user, &entry)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues(entry.APIID)
	ctx.SetPath("/v1/entries/:entryID")

	c.NoError(c.controller.GetEntry(ctx))
	c.Equal(http.StatusOK, rec.Code)

	var sEntry models.Entry
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &sEntry))

	c.Equal(entry.Title, sEntry.Title)
}

func (c *EntriesControllerSuite) TestGetUnknownEntry() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("bogus")
	ctx.SetPath("/v1/entries/:entryID")

	c.EqualError(
		c.controller.GetEntry(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *EntriesControllerSuite) TestGetEntries() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user, &feed)

	entry := models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "example",
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user, &entry)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetPath("/v1/entries")

	c.NoError(c.controller.GetEntries(ctx))

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &entries))

	c.Require().Len(entries.Entries, 1)
	c.Equal(entry.Title, entries.Entries[0].Title)
}

func (c *EntriesControllerSuite) TestMarkEntry() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user, &feed)

	entry := models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "example",
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user, &entry)

	entries, _ := c.entriesRepo.List(c.user, "", 2, true, models.MarkerRead)
	c.Empty(entries)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues(entry.APIID)
	ctx.SetPath("/v1/entries/:entryID/mark")

	c.NoError(c.controller.MarkEntry(ctx))
}

func (c *EntriesControllerSuite) TestMarkUnknownEntry() {
	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("bogus")
	ctx.SetPath("/v1/entries/:entryID/mark")

	c.EqualError(
		c.controller.MarkEntry(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *EntriesControllerSuite) TestMarkAllEntries() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user, &feed)

	entry := models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "example",
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user, &entry)

	entries, _ := c.entriesRepo.List(c.user, "", 2, true, models.MarkerRead)
	c.Empty(entries)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetPath("/v1/entries/mark")

	c.NoError(c.controller.MarkAllEntries(ctx))
}

func (c *EntriesControllerSuite) TestGetEntryStats() {
	req := httptest.NewRequest(echo.PUT, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, *c.user)

	ctx.SetPath("/v1/entries/stats")

	c.NoError(c.controller.GetEntryStats(ctx))

	var stats models.Stats
	c.NoError(json.Unmarshal(rec.Body.Bytes(), &stats))
}

func (c *EntriesControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true
	c.e.Logger.SetLevel(log.OFF)

	c.user = &models.User{
		APIID: utils.CreateAPIID(),
	}

	c.db = sql.NewDB("sqlite3", ":memory:")
	sql.NewUsers(c.db).Create(c.user)

	c.feedsRepo = sql.NewFeeds(c.db)
	c.entriesRepo = sql.NewEntries(c.db)
	c.controller = NewEntriesController(services.NewEntriesService(c.entriesRepo), c.e)
}

func (c *EntriesControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}

func TestEntriesControllerSuite(t *testing.T) {
	suite.Run(t, new(EntriesControllerSuite))
}
