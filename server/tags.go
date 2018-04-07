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
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	tag := models.Tag{}
	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, found := userDB.TagWithName(tag.Name); found {
		return c.JSON(http.StatusConflict, ErrorResp{
			Message: "Tag already exists",
		})
	}

	tag = userDB.NewTag(tag.Name)

	return c.JSON(http.StatusCreated, tag)
}

// GetTags returns a list of Tags owned by a user
func (s *Server) GetTags(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	tags := userDB.Tags()

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	return c.JSON(http.StatusOK, Tags{
		Tags: tags,
	})
}

// DeleteTag with id
func (s *Server) DeleteTag(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	tagID := c.Param("tagID")

	if _, found := userDB.TagWithAPIID(tagID); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	err := userDB.DeleteTag(tagID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Tag could no be deleted",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// EditTag with id
func (s *Server) EditTag(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	tag := models.Tag{}
	tagID := c.Param("tagID")

	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	err := userDB.EditTagName(tagID, tag.Name)
	if err == database.ErrModelNotFound {
		return c.JSON(http.StatusNotFound, ErrorResp{
			"Tag does not exist",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// TagEntries adds a Tag with tagID to a list of entries
func (s *Server) TagEntries(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	tag, found := userDB.TagWithAPIID(c.Param("tagID"))
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

	if _, found := userDB.TagWithAPIID(tag.APIID); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	err := userDB.TagEntries(tag.APIID, entryIds.Entries)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Tag entries could no be fetched",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetTag with id
func (s *Server) GetTag(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	tag, found := userDB.TagWithAPIID(c.Param("tagID"))
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
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	tag, found := userDB.TagWithAPIID(c.Param("tagID"))
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Tag does not exist",
		})
	}

	withMarker := models.MarkerFromString(params.Marker)
	if withMarker == models.None {
		withMarker = models.Any
	}

	entries := userDB.EntriesFromTag(tag.APIID, withMarker, true)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	return c.JSON(http.StatusOK, Entries{
		Entries: entries,
	})
}
