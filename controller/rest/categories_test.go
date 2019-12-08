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
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	CategoriesControllerSuite struct {
		suite.Suite

		controller *CategoriesController
		e          *echo.Echo
		db         *sql.DB
		user       *models.User
	}
)

func (c *CategoriesControllerSuite) TestNewCategory() {
	ctg := `{ "name": "new" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(ctg))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.NoError(c.controller.NewCategory(ctx))
	c.Equal(http.StatusCreated, rec.Code)
}

func (c *CategoriesControllerSuite) TestNewConflictingCategory() {
	ctg := `{ "name": "test" }`

	_, err := c.controller.categories.New("test", c.user.ID)
	c.NoError(err)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(ctg))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	err = c.controller.NewCategory(ctx)

	c.EqualError(
		err,
		echo.NewHTTPError(http.StatusConflict).Error(),
	)
}

func (c *CategoriesControllerSuite) TestNewCategoryWithBadInput() {
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.EqualError(
		echo.NewHTTPError(http.StatusBadRequest),
		c.controller.NewCategory(ctx).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategory() {
	ctg, err := c.controller.categories.New("Test", c.user.ID)
	c.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories/:categoryID")
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	c.NoError(c.controller.GetCategory(ctx))
	c.Equal(http.StatusOK, rec.Code)

	var sCtg models.Category
	c.NoError(json.Unmarshal(rec.Body.Bytes(), &sCtg))
	c.Equal(ctg.Name, sCtg.Name)
}

func (c *CategoriesControllerSuite) TestGetMissingCategory() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories/:categoryID")
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.GetCategory(ctx).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategories() {
	ctg, err := c.controller.categories.New("Test", c.user.ID)
	c.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.NoError(c.controller.GetCategories(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type ctgs struct {
		Categories     []models.Category `json:"categories"`
		ContinuationID string            `json:"continuationID"`
	}

	var categories ctgs
	c.NoError(json.Unmarshal(rec.Body.Bytes(), &categories))
	c.Require().Len(categories.Categories, 1)
	c.Equal(ctg.Name, categories.Categories[0].Name)
}

func (c *CategoriesControllerSuite) TestGetCategoryFeeds() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Subscription: "http://localhost:9090",
		Title:        "example",
		Category:     ctg,
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID/feeds")

	c.NoError(c.controller.GetCategoryFeeds(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	var ctgFeeds feeds
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &ctgFeeds))

	c.Len(ctgFeeds.Feeds, 1)
}

func (c *CategoriesControllerSuite) TestEditCategory() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name": "gopher"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID")

	c.NoError(c.controller.EditCategory(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *CategoriesControllerSuite) TestAppendFeeds() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Subscription: "http://localhost:9090",
		Title:        "example",
		Category:     ctg,
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	feeds := fmt.Sprintf(`{ "feeds": ["%s"] }`, feed.ID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(feeds))
	req.Header.Set("Content-Type", "application/json")

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID/feeds")

	c.NoError(c.controller.AppendCategoryFeeds(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *CategoriesControllerSuite) TestDeleteCategory() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID")

	c.NoError(c.controller.DeleteCategory(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *CategoriesControllerSuite) TestMarkCategory() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Subscription: "http://localhost:9090",
		Title:        "example",
		Category:     ctg,
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	entry := models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	}

	entriesRepo := sql.NewEntries(c.db)
	entriesRepo.Create(c.user.ID, &entry)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID/mark")

	c.NoError(c.controller.MarkCategory(ctx))
	c.Equal(http.StatusNoContent, rec.Code)

	entries, _ := entriesRepo.List(c.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
		Newest:         true,
		Marker:         models.MarkerRead,
	})
	c.Require().Len(entries, 1)
}

func (c *CategoriesControllerSuite) TestMarkUnknownCategory() {
	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/mark")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.MarkCategory(ctx).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryEntries() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Subscription: "http://localhost:9090",
		Title:        "example",
		Category:     ctg,
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	}

	entriesRepo := sql.NewEntries(c.db)
	entriesRepo.Create(c.user.ID, &entry)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID/entries")

	c.NoError(c.controller.GetCategoryEntries(ctx))

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &entries))

	c.Require().Len(entries.Entries, 1)
	c.Equal(entry.Title, entries.Entries[0].Title)
}

func (c *CategoriesControllerSuite) TestGetUnknownCategoryEntries() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/entries")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.GetCategoryEntries(ctx).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryStats() {
	ctg, err := c.controller.categories.New("test", c.user.ID)
	c.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID/stats")

	c.NoError(c.controller.GetCategoryStats(ctx))

	var stats models.Stats
	c.NoError(json.Unmarshal(rec.Body.Bytes(), &stats))
}

func (c *CategoriesControllerSuite) TestGetUnknownCategoryStats() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/stats")

	c.EqualError(
		c.controller.GetCategoryStats(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *CategoriesControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.db = sql.NewDB("sqlite3", ":memory:")
	ctgsRepo := sql.NewCategories(c.db)
	entriesRepo := sql.NewEntries(c.db)
	sql.NewUsers(c.db).Create(c.user)

	c.controller = NewCategoriesController(services.NewCategoriesService(ctgsRepo, entriesRepo), c.e)
}

func (c *CategoriesControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}
func TestCategoriesControllerSuite(t *testing.T) {
	suite.Run(t, new(CategoriesControllerSuite))
}
