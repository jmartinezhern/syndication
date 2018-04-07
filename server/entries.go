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
	"github.com/varddum/syndication/models"
)

// GetEntry with id
func (s *Server) GetEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	entryID := c.Param("entryID")

	entry, found := s.db.EntryWithAPIID(entryID, &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Entry does not exist",
		})
	}

	return c.JSON(http.StatusOK, entry)
}

// GetEntries returns a list of entries that belong to a user
func (s *Server) GetEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries := s.db.Entries(convertOrderByParamToValue(params.OrderBy),
		markedAs,
		&user)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	return c.JSON(http.StatusOK, Entries{
		Entries: entries,
	})
}

// MarkEntry applies a Marker to an Entry
func (s *Server) MarkEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	entryID := c.Param("entryID")

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.None {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	if _, found := s.db.EntryWithAPIID(entryID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Entry does not exist",
		})
	}

	err := s.db.MarkEntry(entryID, marker, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Entry could not be marked",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetStatsForEntries provides statistics related to Entries
func (s *Server) GetStatsForEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	return c.JSON(http.StatusOK, s.db.Stats(&user))
}
