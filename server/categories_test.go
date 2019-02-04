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
)

func (t *ServerTestSuite) TestNewCategory() {
	ctg := `{ "name": "Test" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(ctg))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/categories")

	t.NoError(t.server.NewCategory(c))
	t.Equal(http.StatusCreated, t.rec.Code)
}

func (t *ServerTestSuite) TestNewConflictingCategory() {
	ctg := `{ "name": "test" }`

	database.NewCategory("test", t.user)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(ctg))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/categories")

	err := t.server.NewCategory(c)

	t.EqualError(
		err,
		echo.NewHTTPError(http.StatusConflict).Error(),
	)
}

func (t *ServerTestSuite) TestNewCategoryWithBadInput() {
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/categories")

	t.EqualError(
		echo.NewHTTPError(http.StatusBadRequest),
		t.server.NewCategory(c).Error(),
	)
}

func (t *ServerTestSuite) TestGetCategory() {
	ctg := database.NewCategory("Test", t.user)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/categories/:categoryID")
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	t.NoError(t.server.GetCategory(c))
	t.Equal(http.StatusOK, t.rec.Code)

	var sCtg models.Category
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), &sCtg))
	t.Equal(ctg.Name, sCtg.Name)
}

func (t *ServerTestSuite) TestGetMissingCategory() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/categories/:categoryID")
	c.SetParamNames("categoryID")
	c.SetParamValues("bogus")

	t.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		t.server.GetCategory(c).Error(),
	)
}

func (t *ServerTestSuite) TestGetCategories() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/categories")

	t.NoError(t.server.GetCategories(c))
	t.Equal(http.StatusOK, t.rec.Code)

	type ctgs struct {
		Categories []models.Category `json:"categories"`
	}

	var categories ctgs
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), &categories))
	t.Len(categories.Categories, 1)
	t.Equal("uncategorized", categories.Categories[0].Name)
}

func (t *ServerTestSuite) TestGetCategoryFeeds() {
	ctg := database.NewCategory("test", t.user)
	database.NewFeedWithCategory("example", "example.com", ctg.APIID, t.user)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID/feeds")

	t.NoError(t.server.GetCategoryFeeds(c))
	t.Equal(http.StatusOK, t.rec.Code)

	type feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	var ctgFeeds feeds
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &ctgFeeds))

	t.Len(ctgFeeds.Feeds, 1)
}

func (t *ServerTestSuite) TestEditCategory() {
	ctg := database.NewCategory("test", t.user)

	mdfdCtgJSON := `{"name": "gopher"}`

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(mdfdCtgJSON))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID")

	t.NoError(t.server.EditCategory(c))
	t.Equal(http.StatusOK, t.rec.Code)
}

func (t *ServerTestSuite) TestAppendFeeds() {
	ctg := database.NewCategory("test", t.user)
	feed := database.NewFeed("example", "example.com", t.user)

	feeds := fmt.Sprintf(`{ "feeds": ["%s"] }`, feed.APIID)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(feeds))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID/feeds")

	t.NoError(t.server.AppendCategoryFeeds(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}

func (t *ServerTestSuite) TestDeleteCategory() {
	ctg := database.NewCategory("test", t.user)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID")

	t.NoError(t.server.DeleteCategory(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}

func (t *ServerTestSuite) TestMarkCategory() {
	ctg := database.NewCategory("test", t.user)

	feed, err := database.NewFeedWithCategory(
		"Example", "example.com", ctg.APIID, t.user,
	)
	t.Require().NoError(err)

	database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)

	t.Require().Len(database.Entries(true, models.MarkerRead, t.user), 0)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID/mark")

	t.NoError(t.server.MarkCategory(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}

func (t *ServerTestSuite) TestMarkUnknownCategory() {
	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/categories/:categoryID/mark")

	t.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		t.server.MarkCategory(c).Error(),
	)
}

func (t *ServerTestSuite) TestMarkCategoryWithBadMarker() {
	req := httptest.NewRequest(echo.PUT, "/?as=bogus", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/categories/:categoryID/mark")

	t.EqualError(
		echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required"),
		t.server.MarkCategory(c).Error(),
	)
}

func (t *ServerTestSuite) TestGetCategoryEntries() {
	ctg := database.NewCategory("test", t.user)

	feed, err := database.NewFeedWithCategory(
		"Example",
		"example.com",
		ctg.APIID,
		t.user)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID/mark")

	t.NoError(t.server.GetCategoryEntries(c))

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &entries))

	t.Len(entries.Entries, 1)
	t.Equal(entries.Entries[0].Title, entry.Title)
}

func (t *ServerTestSuite) TestGetUnknownCategoryEntries() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/categories/:categoryID/entries")

	t.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		t.server.GetCategoryEntries(c).Error(),
	)
}

func (t *ServerTestSuite) TestGetCategoryStats() {
	ctg := database.NewCategory("test", t.user)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues(ctg.APIID)

	c.SetPath("/v1/categories/:categoryID/stats")

	t.NoError(t.server.GetCategoryStats(c))

	var stats models.Stats
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), &stats))
}

func (t *ServerTestSuite) TestGetUnknownCategoryStats() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("categoryID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/categories/:categoryID/stats")

	t.EqualError(
		t.server.GetCategoryStats(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}
