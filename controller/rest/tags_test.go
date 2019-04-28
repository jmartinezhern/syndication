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
	TagsControllerSuite struct {
		suite.Suite

		e           *echo.Echo
		db          *sql.DB
		tagsRepo    repo.Tags
		entriesRepo repo.Entries
		user        *models.User
		controller  *TagsController
	}
)

func (c *TagsControllerSuite) TestNewTag() {
	tag := `{ "name": "Test" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(tag))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/tags")

	c.NoError(c.controller.NewTag(ctx))
	c.Equal(http.StatusCreated, rec.Code)
}

func (c *TagsControllerSuite) TestNewConflictingTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`{ "name": "Test" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/tags")

	c.EqualError(
		c.controller.NewTag(ctx),
		echo.NewHTTPError(http.StatusConflict, "tag with name "+tag.Name+" already exists").Error(),
	)
}

func (c *TagsControllerSuite) TestGetTags() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/tags")

	c.NoError(c.controller.GetTags(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	var tags Tags
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &tags))

	c.Require().Len(tags.Tags, 1)
	c.Equal(tag.Name, tags.Tags[0].Name)
}

func (c *TagsControllerSuite) TestDeleteTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tag.ID)

	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.DeleteTag(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *TagsControllerSuite) TestDeleteUnknownTag() {
	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/tags/:tagID")

	c.EqualError(
		c.controller.DeleteTag(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *TagsControllerSuite) TestEditTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	mdfTagJSON := `{"name": "gopher"}`

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(mdfTagJSON))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tag.ID)

	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.UpdateTag(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *TagsControllerSuite) TestEditUnknownTag() {
	req := httptest.NewRequest(
		echo.PUT,
		"/",
		strings.NewReader(`{"name" : "bogus" }`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/tags/:tagID")

	c.EqualError(
		c.controller.UpdateTag(ctx),
		echo.NewHTTPError(http.StatusNotFound, "tag with id bogus not found").Error(),
	)
}

func (c *TagsControllerSuite) TestTagEntries() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "http://example.com",
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test",
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user.ID, &entry)

	req := httptest.NewRequest(
		echo.PUT,
		"/",
		strings.NewReader(fmt.Sprintf(`{
			"entries" :  ["%s"]
		}`, entry.ID)),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tag.ID)

	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.TagEntries(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *TagsControllerSuite) TestTagEntriesWithUnknownTag() {
	req := httptest.NewRequest(
		echo.PUT,
		"/",
		strings.NewReader(`{
			"entries" :  ["foo"]
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/tags/:tagID")

	c.EqualError(
		c.controller.TagEntries(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *TagsControllerSuite) TestGetTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tag.ID)

	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.GetTag(ctx))

	var sTag models.Tag
	c.NoError(json.Unmarshal(rec.Body.Bytes(), &sTag))
	c.Equal(tag.Name, sTag.Name)
}

func (c *TagsControllerSuite) TestGetUnknownTag() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/tags/:tagID")

	c.EqualError(
		c.controller.GetTag(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *TagsControllerSuite) TestGetEntriesFromTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	c.tagsRepo.Create(c.user.ID, &tag)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "http://example.com",
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	entry := models.Entry{
		ID:    utils.CreateID(),
		Title: "Test",
		Feed:  feed,
	}
	c.entriesRepo.Create(c.user.ID, &entry)

	err := c.entriesRepo.TagEntries(c.user.ID, tag.ID, []string{entry.ID})
	c.NoError(err)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tag.ID)

	ctx.SetPath("/v1/tags/:tagID/entries")

	c.NoError(c.controller.GetEntriesFromTag(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &entries))
	c.Require().Len(entries.Entries, 1)
	c.Equal(entry.Title, entries.Entries[0].Title)
	c.Equal(entry.ID, entries.Entries[0].ID)
}

func (c *TagsControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.db = sql.NewDB("sqlite3", ":memory:")
	sql.NewUsers(c.db).Create(c.user)

	c.tagsRepo = sql.NewTags(c.db)
	c.entriesRepo = sql.NewEntries(c.db)
	c.controller = NewTagsController(services.NewTagsService(c.tagsRepo, c.entriesRepo), c.e)
}

func (c *TagsControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}
func TestTagsControllerSuite(t *testing.T) {
	suite.Run(t, new(TagsControllerSuite))
}
