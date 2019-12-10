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
	"bufio"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/jmartinezhern/syndication/services"
)

type (
	Importers = map[string]services.Importer

	ImporterController struct {
		Controller

		importers Importers
	}
)

func NewImporterController(importers Importers, e *echo.Echo) *ImporterController {
	v1 := e.Group("v1")
	controller := ImporterController{
		Controller{
			e,
		},
		importers,
	}
	v1.POST("/import", controller.Import)

	return &controller
}

// Import feeds and categories into Syndication.
// Content-Type header must be set to a supported MIME type.
// The current supported formats are:
//    - OPML (application/xml)
func (s *ImporterController) Import(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	contLength := c.Request().ContentLength
	if contLength <= 0 {
		return echo.NewHTTPError(http.StatusNoContent)
	}

	contType := c.Request().Header.Get("Content-Type")
	data := make([]byte, contLength)
	_, err := bufio.NewReader(c.Request().Body).Read(data)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not read request body")
	}

	if contType == "" && contLength > 0 {
		contType = http.DetectContentType(data)
	}

	if val, ok := s.importers[contType]; ok {
		err = val.Import(data, userID)
	} else {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "could not parse input")
	}

	return c.NoContent(http.StatusNoContent)
}
