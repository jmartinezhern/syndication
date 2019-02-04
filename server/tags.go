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
	"github.com/jmartinezhern/syndication/usecases"
	"net/http"

	"github.com/labstack/echo"
	"github.com/jmartinezhern/syndication/models"
)

// NewTag creates a new Tag
func (s *Server) NewTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag := models.Tag{}
	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newTag, err := s.tags.New(tag.Name, user)
	if err == usecases.ErrTagConflicts {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, newTag)
}

// GetTags returns a list of Tags owned by a user
func (s *Server) GetTags(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"tags": s.tags.Tags(user),
	})
}

// DeleteTag with id
func (s *Server) DeleteTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	err := s.tags.Delete(c.Param("tagID"), user)
	if err == usecases.ErrTagNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// EditTag with id
func (s *Server) EditTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag := models.Tag{}

	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newTag, err := s.tags.Edit(c.Param("tagID"), tag, user)
	if err == usecases.ErrTagNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, newTag)
}

// TagEntries adds a Tag with tagID to a list of entries
func (s *Server) TagEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	entryIds := new(EntryIds)
	if err := c.Bind(entryIds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	err := s.tags.Apply(c.Param("tagID"), entryIds.Entries, user)
	if err == usecases.ErrTagNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetTag with id
func (s *Server) GetTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag, found := s.tags.Tag(c.Param("tagID"), user)
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, tag)
}

// GetEntriesFromTag returns a list of Entries
// that are tagged by a Tag with ID
func (s *Server) GetEntriesFromTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	marker := models.MarkerFromString(params.Marker)
	if marker == models.MarkerNone {
		marker = models.MarkerAny
	}

	entries, err := s.tags.Entries(
		c.Param("tagID"),
		marker,
		convertOrderByParamToValue(params.OrderBy),
		user)
	if err == usecases.ErrTagNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries": entries,
	})
}
