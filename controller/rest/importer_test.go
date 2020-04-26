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

package rest_test

import (
	"errors"
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
	ImporterControllerSuite struct {
		suite.Suite

		ctrl         *gomock.Controller
		mockImporter *services.MockImporter

		controller *rest.ImporterController
		e          *echo.Echo
		user       *models.User
	}
)

func (c *ImporterControllerSuite) TestImport() {
	c.mockImporter.EXPECT().Import(gomock.Any(), gomock.Eq(c.user.ID)).Return(nil)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`<xml></xml>`))
	req.Header.Set("Content-Type", "text/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/import")

	c.NoError(c.controller.Import(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *ImporterControllerSuite) TestImportEmptyRequest() {
	req := httptest.NewRequest(echo.POST, "/", nil)
	req.Header.Set("Content-Type", "text/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/import")

	c.NoError(c.controller.Import(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *ImporterControllerSuite) TestImportDetectContentType() {
	c.mockImporter.EXPECT().Import(gomock.Any(), gomock.Eq(c.user.ID)).Return(nil)

	req := httptest.NewRequest(echo.POST, "/",
		strings.NewReader(`<?xml version="1.0"?><opml></opml>`))

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/import")

	c.NoError(c.controller.Import(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *ImporterControllerSuite) TestImportInternalError() {
	c.mockImporter.EXPECT().Import(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	req := httptest.NewRequest(echo.POST, "/",
		strings.NewReader(`<?xml version="1.0"?><opml></opml>`))
	req.Header.Set("Content-Type", "text/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/import")

	c.EqualError(
		c.controller.Import(ctx),
		echo.NewHTTPError(http.StatusBadRequest, "could not parse input").Error(),
	)
}

func (c *ImporterControllerSuite) TestImportUnsupportedContentType() {
	req := httptest.NewRequest(echo.POST, "/",
		strings.NewReader(`<?xml version="1.0"?><opml></opml>`))
	req.Header.Set("Content-Type", "text/bogus")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/import")

	c.EqualError(
		c.controller.Import(ctx),
		echo.NewHTTPError(http.StatusUnsupportedMediaType).Error(),
	)
}

func (c *ImporterControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.mockImporter = services.NewMockImporter(c.ctrl)

	importers := rest.Importers{
		"text/xml": c.mockImporter,
	}

	c.controller = rest.NewImporterController(importers, c.e)
}

func (c *ImporterControllerSuite) TearDownTest() {
	c.ctrl.Finish()
}

func TestImporterControllerSuite(t *testing.T) {
	suite.Run(t, new(ImporterControllerSuite))
}
