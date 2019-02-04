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
	"net/http"

	"github.com/labstack/echo"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/usecases"
)

// Export feeds and categories out of Syndication.
// Accept header must be set to a supported MIME type.
// The current supported formats are:
//    - OPML (application/xml)
func (s *Server) Export(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	contType := c.Request().Header.Get("Accept")

	switch contType {
	case "application/xml":
		exporter := usecases.OPMLExporter{}
		data, err := exporter.Export(user)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		return c.XMLBlob(http.StatusOK, data)
	default:
		return echo.NewHTTPError(http.StatusBadRequest)
	}
}
