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
	"errors"
	"net/http"
	"net/http/httptest"
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
	EntriesControllerSuite struct {
		suite.Suite

		ctrl        *gomock.Controller
		mockEntries *services.MockEntries

		controller *rest.EntriesController
		e          *echo.Echo
		user       *models.User
	}
)

func (c *EntriesControllerSuite) TestGetEntry() {
	entryID := utils.CreateID()

	c.mockEntries.EXPECT().Entry(gomock.Eq(c.user.ID), gomock.Eq(entryID)).Return(models.Entry{ID: entryID}, nil)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues(entryID)
	ctx.SetPath("/v1/entries/:entryID")

	c.NoError(c.controller.GetEntry(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *EntriesControllerSuite) TestGetUnknownEntry() {
	c.mockEntries.EXPECT().Entry(gomock.Any(), gomock.Any()).Return(models.Entry{}, services.ErrEntryNotFound)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("bogus")
	ctx.SetPath("/v1/entries/:entryID")

	c.EqualError(
		c.controller.GetEntry(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *EntriesControllerSuite) TestGetEntryInternalError() {
	c.mockEntries.EXPECT().Entry(gomock.Any(), gomock.Any()).Return(models.Entry{}, errors.New("error"))

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("bogus")
	ctx.SetPath("/v1/entries/:entryID")

	c.EqualError(
		c.controller.GetEntry(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *EntriesControllerSuite) TestGetEntries() {
	page := models.Page{
		Count:  1,
		Newest: true,
		Marker: models.MarkerAny,
	}

	c.mockEntries.EXPECT().Entries(gomock.Eq(c.user.ID), gomock.Eq(page)).Return([]models.Entry{
		{
			ID: utils.CreateID(),
		},
	}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/entries")

	c.NoError(c.controller.GetEntries(ctx))
}

func (c *EntriesControllerSuite) TestGetEntriesBadRequest() {
	req := httptest.NewRequest(echo.GET, "/?count=true", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/entries")

	c.EqualError(
		c.controller.GetEntries(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *EntriesControllerSuite) TestMarkEntry() {
	entryID := utils.CreateID()

	c.mockEntries.EXPECT().Mark(gomock.Eq(c.user.ID), gomock.Eq(entryID), gomock.Eq(models.MarkerRead)).Return(nil)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues(entryID)
	ctx.SetPath("/v1/entries/:entryID/mark")

	c.NoError(c.controller.MarkEntry(ctx))
}

func (c *EntriesControllerSuite) TestMarkUnknownEntry() {
	c.mockEntries.EXPECT().Mark(gomock.Any(), gomock.Any(), gomock.Any()).Return(services.ErrEntryNotFound)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("bogus")
	ctx.SetPath("/v1/entries/:entryID/mark")

	c.EqualError(
		c.controller.MarkEntry(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *EntriesControllerSuite) TestMarkEntryBadRequest() {
	req := httptest.NewRequest(echo.PUT, "/?as=", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("id")
	ctx.SetPath("/v1/entries/:entryID/mark")

	c.EqualError(
		c.controller.MarkEntry(ctx),
		echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required").Error(),
	)
}

func (c *EntriesControllerSuite) TestMarkEntryInternalError() {
	c.mockEntries.EXPECT().Mark(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("error"))

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("id")
	ctx.SetPath("/v1/entries/:entryID/mark")

	c.EqualError(
		c.controller.MarkEntry(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *EntriesControllerSuite) TestMarkAllEntries() {
	c.mockEntries.EXPECT().MarkAll(gomock.Eq(c.user.ID), models.MarkerRead)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/entries/mark")

	c.NoError(c.controller.MarkAllEntries(ctx))
}

func (c *EntriesControllerSuite) TestMarkAllEntriesBadRequest() {
	req := httptest.NewRequest(echo.PUT, "/?as=", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("entryID")
	ctx.SetParamValues("id")
	ctx.SetPath("/v1/entries/mark")

	c.EqualError(
		c.controller.MarkAllEntries(ctx),
		echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required").Error(),
	)
}

func (c *EntriesControllerSuite) TestGetEntryStats() {
	c.mockEntries.EXPECT().Stats(gomock.Eq(c.user.ID)).Return(models.Stats{})

	req := httptest.NewRequest(echo.PUT, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/entries/stats")

	c.NoError(c.controller.GetEntryStats(ctx))
}

func (c *EntriesControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.mockEntries = services.NewMockEntries(c.ctrl)

	c.controller = rest.NewEntriesController(c.mockEntries, c.e)
}

func (c *EntriesControllerSuite) TearDownTest() {
	c.ctrl.Finish()
}

func TestEntriesControllerSuite(t *testing.T) {
	suite.Run(t, new(EntriesControllerSuite))
}
