/*
 *   Copyright (C) 2021. Jorge Martinez Hernandez
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU Affero General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU Affero General Public License for more details.
 *
 *   You should have received a copy of the GNU Affero General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package rest_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	TagsControllerSuite struct {
		suite.Suite

		ctrl     *gomock.Controller
		mockTags *services.MockTags

		controller *rest.TagsController
		e          *echo.Echo
		user       *models.User
	}
)

func (c *TagsControllerSuite) TestNewTag() {
	c.mockTags.EXPECT().New(gomock.Eq(c.user.ID), gomock.Eq("Test")).Return(models.Tag{ID: utils.CreateID()}, nil)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`{ "name": "Test" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/tags")

	c.NoError(c.controller.NewTag(ctx))
	c.Equal(http.StatusCreated, rec.Code)
}

func (c *TagsControllerSuite) TestNewConflictingTag() {
	c.mockTags.EXPECT().New(gomock.Any(), gomock.Any()).Return(models.Tag{}, services.ErrTagConflicts)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`{ "name": "Test" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/tags")

	c.EqualError(
		c.controller.NewTag(ctx),
		echo.NewHTTPError(http.StatusConflict, "tag with name Test already exists").Error(),
	)
}

func (c *TagsControllerSuite) TestGetTags() {
	page := models.Page{
		Count: 1,
	}

	c.mockTags.EXPECT().List(gomock.Eq(c.user.ID), gomock.Eq(page)).Return([]models.Tag{}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/tags")

	c.NoError(c.controller.GetTags(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *TagsControllerSuite) TestDeleteTag() {
	tagID := utils.CreateID()

	c.mockTags.EXPECT().Delete(gomock.Eq(c.user.ID), gomock.Eq(tagID)).Return(nil)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tagID)
	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.DeleteTag(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *TagsControllerSuite) TestDeleteUnknownTag() {
	c.mockTags.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(services.ErrTagNotFound)

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
	tagID := utils.CreateID()

	c.mockTags.EXPECT().
		Update(gomock.Eq(c.user.ID), gomock.Eq(tagID), gomock.Eq("gopher")).
		Return(models.Tag{ID: tagID, Name: "gopher"}, nil)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name": "gopher"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tagID)
	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.UpdateTag(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *TagsControllerSuite) TestEditUnknownTag() {
	c.mockTags.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Tag{}, services.ErrTagNotFound)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name" : "bogus" }`))
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

func (c *TagsControllerSuite) TestEditTagConflicts() {
	c.mockTags.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any()).Return(models.Tag{}, services.ErrTagConflicts)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name" : "bogus" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/tags/:tagID")

	c.EqualError(
		c.controller.UpdateTag(ctx),
		echo.NewHTTPError(http.StatusConflict, "tag with name bogus already exists").Error(),
	)
}

func (c *TagsControllerSuite) TestTagEntries() {
	entryID := utils.CreateID()
	tagID := utils.CreateID()

	c.mockTags.EXPECT().Apply(gomock.Eq(c.user.ID), gomock.Eq(tagID), gomock.Eq([]string{entryID})).Return(nil)

	req := httptest.NewRequest(echo.PUT, "/",
		strings.NewReader(fmt.Sprintf(`{ "entries" : ["%s"] }`, entryID)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tagID)

	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.TagEntries(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *TagsControllerSuite) TestTagEntriesWithUnknownTag() {
	c.mockTags.EXPECT().Apply(gomock.Any(), gomock.Any(), gomock.Any()).Return(services.ErrTagNotFound)

	req := httptest.NewRequest(echo.PUT, "/",
		strings.NewReader(`{ "entries" :  ["foo"] }`))
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
	tagID := utils.CreateID()

	c.mockTags.EXPECT().Tag(gomock.Eq(c.user.ID), gomock.Eq(tagID)).Return(models.Tag{ID: tagID}, true)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tagID)

	ctx.SetPath("/v1/tags/:tagID")

	c.NoError(c.controller.GetTag(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *TagsControllerSuite) TestGetUnknownTag() {
	c.mockTags.EXPECT().Tag(gomock.Any(), gomock.Any()).Return(models.Tag{}, false)

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
	tagID := utils.CreateID()

	page := models.Page{
		Count:  1,
		Newest: true,
		Marker: models.MarkerAny,
	}

	c.mockTags.EXPECT().
		Entries(gomock.Eq(c.user.ID), gomock.Eq(page)).
		Return([]models.Entry{
			{
				ID: utils.CreateID(),
			},
		}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("tagID")
	ctx.SetParamValues(tagID)
	ctx.SetPath("/v1/tags/:tagID/entries")

	c.NoError(c.controller.GetEntriesFromTag(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *TagsControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.mockTags = services.NewMockTags(c.ctrl)

	c.controller = rest.NewTagsController(c.mockTags, c.e)
}

func (c *TagsControllerSuite) TearDownTest() {
	c.ctrl.Finish()
}

func TestTagsControllerSuite(t *testing.T) {
	suite.Run(t, new(TagsControllerSuite))
}
