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
	"bufio"
	"net/http"

	"github.com/labstack/echo"

	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/usecases"
)

// Import feeds and categories into Syndication.
// Content-Type header must be set to a supported MIME type.
// The current supported formats are:
//    - OPML (application/xml)
func (s *Server) Import(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

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

	switch contType {
	case "application/xml":
		importer := usecases.OPMLImporter{}
		err = importer.Import(data, user)
	default:
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
