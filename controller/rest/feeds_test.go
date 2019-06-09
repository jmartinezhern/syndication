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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	FeedsControllerSuite struct {
		suite.Suite

		e           *echo.Echo
		db          *sql.DB
		feedsRepo   repo.Feeds
		ctgsRepo    repo.Categories
		entriesRepo repo.Entries
		user        *models.User
		controller  *FeedsController
	}
)

func (c *FeedsControllerSuite) TestNewFeed() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<rss></rss>")
	}))
	defer ts.Close()

	feed := fmt.Sprintf(`{ "title": "Example", "subscription": "%s" }`, ts.URL)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(feed))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/feeds")

	c.NoError(c.controller.NewFeed(ctx))
	c.Equal(http.StatusCreated, rec.Code)
}

func (c *FeedsControllerSuite) TestUnreachableNewFeed() {
	feed := `{ "title": "Example", "subscription": "bogus" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(feed))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/feeds")

	c.EqualError(c.controller.NewFeed(ctx), echo.NewHTTPError(http.StatusBadRequest, "subscription url is not reachable").Error())
}

func (c *FeedsControllerSuite) TestGetFeeds() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/feeds")

	c.NoError(c.controller.GetFeeds(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	var feeds Feeds
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &feeds))

	c.Len(feeds.Feeds, 1)
	c.Equal(feed.Title, feeds.Feeds[0].Title)
}

func (c *FeedsControllerSuite) TestGetFeed() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feed.ID)

	ctx.SetPath("/v1/feeds/:feedID")

	c.NoError(c.controller.GetFeed(ctx))
	c.Equal(http.StatusOK, rec.Code)

	var sFeed models.Feed
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &sFeed))

	c.Equal(feed.Title, sFeed.Title)
}

func (c *FeedsControllerSuite) TestGetUnknownFeed() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID")

	c.EqualError(
		c.controller.GetFeed(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) TestEditFeed() {
	newFeed := `{ "title": "NewName" }`
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(newFeed))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feed.ID)

	ctx.SetPath("/v1/feeds/:feedID")

	c.NoError(c.controller.EditFeed(ctx))
	c.Equal(http.StatusOK, rec.Code)

	var sFeed models.Feed
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &sFeed))

	c.Equal("NewName", sFeed.Title)
}

func (c *FeedsControllerSuite) TestEditUnknownFeed() {
	newFeed := `{ "title": "title" }`

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(newFeed))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID")

	c.EqualError(
		c.controller.EditFeed(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) TestDeleteFeed() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feed.ID)

	ctx.SetPath("/v1/feeds/:feedID")

	c.NoError(c.controller.DeleteFeed(ctx))

	_, found := c.feedsRepo.FeedWithID(c.user.ID, feed.ID)
	c.False(found)
}

func (c *FeedsControllerSuite) TestDeleteUnknownFeed() {
	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.DeleteFeed(ctx).Error(),
	)
}

func (c *FeedsControllerSuite) TestMarkFeed() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user.ID, &entry)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feed.ID)

	ctx.SetPath("/v1/feeds/:feedID/mark")

	c.NoError(c.controller.MarkFeed(ctx))
	c.Equal(http.StatusNoContent, rec.Code)

	entries, _ := c.entriesRepo.List(c.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
		Newest:         false,
		Marker:         models.MarkerRead,
	})
	c.Len(entries, 1)
}

func (c *FeedsControllerSuite) TestMarkUnknownFeeed() {
	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID/mark")

	c.EqualError(
		c.controller.MarkFeed(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) TestGetFeedEntries() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	entry := models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user.ID, &entry)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feed.ID)
	ctx.SetPath("/v1/feeds/:feedID/entries")

	c.NoError(c.controller.GetFeedEntries(ctx))

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &entries))

	c.Require().Len(entries.Entries, 1)
	c.Equal(entry.Title, entries.Entries[0].Title)
}

func (c *FeedsControllerSuite) TestGetFeedStats() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "http://example.com",
	}
	c.feedsRepo.Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feed.ID)

	ctx.SetPath("/v1/feeds/:feedID/stats")

	c.NoError(c.controller.GetFeedStats(ctx))

	var stats models.Stats
	c.NoError(json.Unmarshal(rec.Body.Bytes(), &stats))
}

func (c *FeedsControllerSuite) TestGetUnknownFeedStats() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID/stats")

	c.EqualError(
		c.controller.GetFeedStats(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.db = sql.NewDB("sqlite3", ":memory:")
	sql.NewUsers(c.db).Create(c.user)

	c.feedsRepo = sql.NewFeeds(c.db)
	c.ctgsRepo = sql.NewCategories(c.db)
	c.entriesRepo = sql.NewEntries(c.db)
	c.controller = NewFeedsController(services.NewFeedsService(c.feedsRepo, c.ctgsRepo, c.entriesRepo), c.e)
}

func (c *FeedsControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}

func TestFeedsControllerSuite(t *testing.T) {
	suite.Run(t, new(FeedsControllerSuite))
}
