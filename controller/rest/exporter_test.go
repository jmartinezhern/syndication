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
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	ExporterControllerSuite struct {
		suite.Suite

		ctrl         *gomock.Controller
		mockExporter *services.MockExporter

		controller *rest.ExporterController
		e          *echo.Echo
		user       *models.User
	}
)

func (c *ExporterControllerSuite) TestOPMLExport() {
	c.mockExporter.EXPECT().Export(gomock.Eq(c.user.ID)).Return([]byte{}, nil)

	req := httptest.NewRequest(echo.GET, "/", nil)
	req.Header.Set("Accept", "application/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/export")

	c.NoError(c.controller.Export(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *ExporterControllerSuite) TestOPMLExportInternalError() {
	c.mockExporter.EXPECT().Export(gomock.Any()).Return([]byte{}, errors.New("error"))

	req := httptest.NewRequest(echo.GET, "/", nil)
	req.Header.Set("Accept", "application/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/export")

	c.EqualError(
		c.controller.Export(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *ExporterControllerSuite) TestOPMLExportBadAcceptHeader() {
	c.mockExporter.EXPECT().Export(gomock.Any()).Return([]byte{}, nil)

	req := httptest.NewRequest(echo.GET, "/", nil)
	req.Header.Set("Accept", "application/bogus")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/export")

	c.EqualError(
		c.controller.Export(ctx),
		echo.NewHTTPError(http.StatusNotAcceptable).Error(),
	)
}

func (c *ExporterControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.mockExporter = services.NewMockExporter(c.ctrl)

	exporters := rest.Exporters{
		"application/xml": c.mockExporter,
	}
	c.controller = rest.NewExporterController(exporters, c.e)
}

func (c *ExporterControllerSuite) TearDownTest() {
}
func TestExporterControllerSuite(t *testing.T) {
	suite.Run(t, new(ExporterControllerSuite))
}
