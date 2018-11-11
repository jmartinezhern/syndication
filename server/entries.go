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
	"github.com/varddum/syndication/usecases"
	"net/http"

	"github.com/labstack/echo"
	"github.com/varddum/syndication/models"
)

type (
	// EntryQueryParams maps query parameters used when GETting entries resources
	EntryQueryParams struct {
		Marker  string `query:"markedAs"`
		Saved   bool   `query:"saved"`
		OrderBy string `query:"orderBy"`
	}
)

// GetEntry with id
func (s *Server) GetEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	entry, err := s.entries.Entry(c.Param("entryID"), user)
	if err == usecases.ErrEntryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
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

	marker := models.MarkerFromString(params.Marker)
	if marker == models.MarkerNone {
		marker = models.MarkerAny
	}

	entries := s.entries.Entries(
		convertOrderByParamToValue(params.OrderBy),
		marker,
		user,
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries": entries,
	})
}

// MarkEntry applies a Marker to an Entry
func (s *Server) MarkEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.MarkerNone {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	err := s.entries.Mark(c.Param("entryID"), marker, user)
	if err == usecases.ErrEntryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkAllEntries applies a Marker to all Entries
func (s *Server) MarkAllEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.MarkerNone {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	s.entries.MarkAll(marker, user)

	return c.NoContent(http.StatusNoContent)
}

// GetEntryStats provides statistics related to Entries
func (s *Server) GetEntryStats(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	return c.JSON(http.StatusOK, s.entries.Stats(user))
}
