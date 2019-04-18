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
	"net/http"

	"github.com/labstack/echo"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

type (
	Exporters = map[string]services.Exporter

	ExporterController struct {
		Controller

		exporters Exporters
	}
)

func NewExporterController(exporters Exporters, e *echo.Echo) *ExporterController {
	v1 := e.Group("v1")

	controller := ExporterController{
		Controller{
			e,
		},
		exporters,
	}

	v1.GET("/export", controller.Export)

	return &controller
}

// Export feeds and categories out of Syndication.
// Accept header must be set to a supported MIME type.
// The current supported formats are:
//    - OPML (application/xml)
func (s *ExporterController) Export(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	contType := c.Request().Header.Get("Accept")

	if exporter, ok := s.exporters[contType]; ok {
		data, err := exporter.Export(&user)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		switch contType {
		case "application/xml":
			return c.XMLBlob(http.StatusOK, data)
		case "application/json":
			return c.JSONBlob(http.StatusOK, data)
		}
	}

	return echo.NewHTTPError(http.StatusNotAcceptable)
}
