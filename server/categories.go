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

// GetCategory with id
func (s *Server) GetCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg, found := s.db.CategoryWithAPIID(c.Param("categoryID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	return c.JSON(http.StatusOK, ctg)
}

// GetCategories returns a list of Categories owned by a user
func (s *Server) GetCategories(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgs := s.db.Categories(&user)

	type Categories struct {
		Categories []models.Category `json:"categories"`
	}

	return c.JSON(http.StatusOK, Categories{
		Categories: ctgs,
	})
}

// GetFeedsFromCategory returns a list of Feeds that belong to a Category
func (s *Server) GetFeedsFromCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg, found := s.db.CategoryWithAPIID(c.Param("categoryID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	feeds := s.db.FeedsFromCategory(ctg.APIID, &user)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	return c.JSON(http.StatusOK, Feeds{
		Feeds: feeds,
	})
}

// NewCategory creates a new Category
func (s *Server) NewCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg := models.Category{}
	if err := c.Bind(&ctg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, found := s.db.CategoryWithName(ctg.Name, &user); found {
		return c.JSON(http.StatusConflict, ErrorResp{
			Message: "Category already exists",
		})
	}

	ctg = s.db.NewCategory(ctg.Name, &user)

	return c.JSON(http.StatusCreated, ctg)
}

// EditCategory with id
func (s *Server) EditCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg := models.Category{}
	ctg.APIID = c.Param("categoryID")

	if err := c.Bind(&ctg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, found := s.db.CategoryWithAPIID(ctg.APIID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	err := s.db.EditCategory(&ctg, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Category could not be edited",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// AddFeedsToCategory adds a Feed to a Category with id
func (s *Server) AddFeedsToCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	type FeedIds struct {
		Feeds []string `json:"feeds"`
	}

	feedIds := new(FeedIds)
	if err := c.Bind(feedIds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, found := s.db.CategoryWithAPIID(ctgID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	for _, id := range feedIds.Feeds {
		err := s.db.ChangeFeedCategory(id, ctgID, &user)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// DeleteCategory with id
func (s *Server) DeleteCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	if _, found := s.db.CategoryWithAPIID(ctgID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	err := s.db.DeleteCategory(ctgID, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Category could not be deleted",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// MarkCategory applies a Marker to a Category
func (s *Server) MarkCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.None {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	if _, found := s.db.CategoryWithAPIID(ctgID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	err := s.db.MarkCategory(ctgID, marker, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Category could not be marked",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetEntriesFromCategory returns a list of Entries
// that belong to a Feed that belongs to a Category
func (s *Server) GetEntriesFromCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	ctg, found := s.db.CategoryWithAPIID(c.Param("categoryID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Category does not exist",
		})
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries := s.db.EntriesFromCategory(ctg.APIID,
		convertOrderByParamToValue(params.OrderBy),
		markedAs,
		&user)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	return c.JSON(http.StatusOK, Entries{
		Entries: entries,
	})
}

// GetStatsForCategory returns statistics related to a Category
func (s *Server) GetStatsForCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	if _, found := s.db.CategoryWithAPIID(ctgID, &user); !found {
		return c.JSON(http.StatusNotFound, ErrorResp{
			Message: "Category does not exist",
		})
	}

	marks := s.db.CategoryStats(ctgID, &user)

	return c.JSON(http.StatusOK, marks)
}
