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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

func (t *ServerTestSuite) TestNewFeed() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "<rss></rss>")
	}))
	defer ts.Close()

	feed := fmt.Sprintf(`{ "title": "Example", "subscription": "%s" }`, ts.URL)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(feed))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/feeds")

	t.NoError(t.server.NewFeed(c))
	t.Equal(http.StatusCreated, t.rec.Code)
}

func (t *ServerTestSuite) TestUnreachableNewFeed() {
	feed := `{ "title": "Example", "subscription": "bogus" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(feed))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/feeds")

	t.EqualError(t.server.NewFeed(c), echo.NewHTTPError(http.StatusBadRequest).Error())
}

func (t *ServerTestSuite) TestGetFeeds() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetPath("/v1/feeds")

	t.NoError(t.server.GetFeeds(c))
	t.Equal(http.StatusOK, t.rec.Code)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	var feeds Feeds
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &feeds))

	t.Len(feeds.Feeds, 1)
	t.Equal(feed.Title, feeds.Feeds[0].Title)
}

func (t *ServerTestSuite) TestGetFeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues(feed.APIID)

	c.SetPath("/v1/feeds/:feedID")

	t.NoError(t.server.GetFeed(c))
	t.Equal(http.StatusOK, t.rec.Code)

	var sFeed models.Feed
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &sFeed))

	t.Equal(feed.Title, sFeed.Title)
}

func (t *ServerTestSuite) TestGetUnknownFeed() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/feeds/:feedID")

	t.EqualError(
		t.server.GetFeed(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestEditFeed() {
	newFeed := `{ "title": "NewName" }`
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(newFeed))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues(feed.APIID)

	c.SetPath("/v1/feeds/:feedID")

	t.NoError(t.server.EditFeed(c))
	t.Equal(http.StatusOK, t.rec.Code)

	var sFeed models.Feed
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &sFeed))

	t.Equal("NewName", sFeed.Title)
}

func (t *ServerTestSuite) TestEditUnkownFeed() {
	newFeed := `{ "title": "NewName" }`

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(newFeed))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/feeds/:feedID")

	t.EqualError(
		t.server.EditFeed(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestDeleteFeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	feeds, _ := database.Feeds("", 5, t.user)

	t.NotEmpty(feeds)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues(feed.APIID)

	c.SetPath("/v1/feeds/:feedID")

	t.NoError(t.server.DeleteFeed(c))
}

func (t *ServerTestSuite) TestDeleteUnknownFeed() {
	req := httptest.NewRequest(echo.DELETE, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/feeds/:feedID")

	t.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		t.server.DeleteFeed(c).Error(),
	)
}

func (t *ServerTestSuite) TestMarkFeeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)

	entries, _ := database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Require().Len(entries, 0)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues(feed.APIID)

	c.SetPath("/v1/feeds/:feedID/mark")

	t.NoError(t.server.MarkFeed(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}

func (t *ServerTestSuite) TestMarkUnknownFeeed() {
	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/feeds/:feedID/mark")

	t.EqualError(
		t.server.MarkFeed(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestGetFeedEntries() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetParamNames("feedID")
	c.SetParamValues(feed.APIID)
	c.SetPath("/v1/feeds/:feedID/entries")

	t.NoError(t.server.GetFeedEntries(c))

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &entries))

	t.Len(entries.Entries, 1)
	t.Equal(entry.Title, entries.Entries[0].Title)
}

func (t *ServerTestSuite) TestGetUnknownFeedEntries() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)

	c.SetParamNames("feedID")
	c.SetParamValues("bogus")
	c.SetPath("/v1/feeds/:feedID/entries")

	t.EqualError(
		t.server.GetFeedEntries(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestGetFeedStats() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Example",
		Subscription: "http://localhost:9090",
	}
	err := database.CreateFeed(&feed, t.unctgCtg.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues(feed.APIID)

	c.SetPath("/v1/feeds/:feedID/stats")

	t.NoError(t.server.GetFeedStats(c))

	var stats models.Stats
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), &stats))
}

func (t *ServerTestSuite) TestGetUnknownFeedStats() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(userContextKey, t.user)
	c.SetParamNames("feedID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/feeds/:feedID/stats")

	t.EqualError(
		t.server.GetFeedStats(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}
