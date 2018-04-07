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
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

// NewTag creates a new Tag
func (s *Server) NewTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag := models.Tag{}
	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, found := s.db.TagWithName(tag.Name, &user); found {
		return c.JSON(http.StatusConflict, ErrorResp{
			Message: "Tag already exists",
		})
	}

	tag = s.db.NewTag(tag.Name, &user)

	return c.JSON(http.StatusCreated, tag)
}

// GetTags returns a list of Tags owned by a user
func (s *Server) GetTags(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tags := s.db.Tags(&user)

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	return c.JSON(http.StatusOK, Tags{
		Tags: tags,
	})
}

// DeleteTag with id
func (s *Server) DeleteTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tagID := c.Param("tagID")

	if _, found := s.db.TagWithAPIID(tagID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	err := s.db.DeleteTag(tagID, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Tag could no be deleted",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// EditTag with id
func (s *Server) EditTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag := models.Tag{}
	tagID := c.Param("tagID")

	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	err := s.db.EditTagName(tagID, tag.Name, &user)
	if err == database.ErrModelNotFound {
		return c.JSON(http.StatusNotFound, ErrorResp{
			"Tag does not exist",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// TagEntries adds a Tag with tagID to a list of entries
func (s *Server) TagEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag, found := s.db.TagWithAPIID(c.Param("tagID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	entryIds := new(EntryIds)
	if err := c.Bind(entryIds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, found := s.db.TagWithAPIID(tag.APIID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	err := s.db.TagEntries(tag.APIID, entryIds.Entries, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Tag entries could no be fetched",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetTag with id
func (s *Server) GetTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag, found := s.db.TagWithAPIID(c.Param("tagID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
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

	tag, found := s.db.TagWithAPIID(c.Param("tagID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	withMarker := models.MarkerFromString(params.Marker)
	if withMarker == models.None {
		withMarker = models.Any
	}

	entries := s.db.EntriesFromTag(tag.APIID, withMarker, true, &user)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	return c.JSON(http.StatusOK, Entries{
		Entries: entries,
	})
}
