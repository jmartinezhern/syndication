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

	"github.com/labstack/echo/v4"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

type (
	EntriesController struct {
		Controller

		entries services.Entries
	}
)

func NewEntriesController(service services.Entries, e *echo.Echo) *EntriesController {
	controller := EntriesController{
		Controller: Controller{
			e,
		},
		entries: service,
	}

	v1 := e.Group("v1")

	v1.GET("/entries", controller.GetEntries)
	v1.GET("/entries/:entryID", controller.GetEntry)
	v1.PUT("/entries/:entryID/mark", controller.MarkEntry)
	v1.PUT("/entries/mark", controller.MarkAllEntries)
	v1.GET("/entries/stats", controller.GetEntryStats)

	return &controller
}

// GetEntry with id
func (s *EntriesController) GetEntry(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	entryID := c.Param("entryID")

	entry, err := s.entries.Entry(entryID, userID)
	if err == services.ErrEntryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, entry)
}

// GetEntries returns a list of entries that belong to a user
func (s *EntriesController) GetEntries(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := new(listEntriesParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	marker := models.MarkerFromString(params.Marker)

	entries, next := s.entries.Entries(
		params.ContinuationID,
		params.Count,
		convertOrderByParamToValue(params.OrderBy),
		marker,
		userID,
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries":        entries,
		"continuationID": next,
	})
}

// MarkEntry applies a Marker to an Entries
func (s *EntriesController) MarkEntry(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	asParam := c.FormValue("as")
	if asParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	marker := models.MarkerFromString(asParam)
	entryID := c.Param("entryID")

	err := s.entries.Mark(entryID, marker, userID)
	if err == services.ErrEntryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkAllEntries applies a Marker to all Entries
func (s *EntriesController) MarkAllEntries(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	asParam := c.FormValue("as")
	if asParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	marker := models.MarkerFromString(asParam)

	s.entries.MarkAll(marker, userID)

	return c.NoContent(http.StatusNoContent)
}

// GetEntryStats provides statistics related to Entries
func (s *EntriesController) GetEntryStats(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	return c.JSON(http.StatusOK, s.entries.Stats(userID))
}
