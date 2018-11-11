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

	"github.com/varddum/syndication/usecases"

	"github.com/labstack/echo"
	"github.com/varddum/syndication/models"
)

// NewCategory creates a new Category
func (s *Server) NewCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg := models.Category{}
	if err := c.Bind(&ctg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newCtg, err := s.categories.New(ctg.Name, user)
	if err == usecases.ErrCategoryConflicts {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, newCtg)
}

// GetCategory with id
func (s *Server) GetCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg, found := s.categories.Category(c.Param("categoryID"), user)
	if found {
		return c.JSON(http.StatusOK, ctg)
	}

	return echo.NewHTTPError(http.StatusNotFound)
}

// GetCategories returns a list of Categories owned by a user
func (s *Server) GetCategories(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": s.categories.Categories(user),
	})
}

// GetCategoryFeeds returns a list of Feeds that belong to a Category
func (s *Server) GetCategoryFeeds(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feeds, err := s.categories.Feeds(c.Param("categoryID"), user)
	if err == usecases.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"feeds": feeds,
	})
}

// EditCategory with id
func (s *Server) EditCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctg := models.Category{}
	ctgID := c.Param("categoryID")

	if err := c.Bind(&ctg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newCtg, err := s.categories.Edit(ctg.Name, ctgID, user)
	if err == usecases.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err == usecases.ErrCategoryProtected {
		return echo.NewHTTPError(http.StatusBadRequest)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, newCtg)
}

// AppendCategoryFeeds adds a Feed to a Category with id
func (s *Server) AppendCategoryFeeds(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	type FeedIds struct {
		Feeds []string `json:"feeds"`
	}

	feeds := new(FeedIds)
	if err := c.Bind(feeds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	s.categories.AddFeeds(ctgID, feeds.Feeds, user)

	return c.NoContent(http.StatusNoContent)
}

// DeleteCategory with id
func (s *Server) DeleteCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	err := s.categories.Delete(c.Param("categoryID"), user)
	if err == usecases.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err == usecases.ErrCategoryProtected {
		return echo.NewHTTPError(http.StatusBadRequest)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkCategory applies a Marker to a Category
func (s *Server) MarkCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.MarkerNone {
		return echo.NewHTTPError(http.StatusBadRequest,
			"'as' parameter is required")
	}

	err := s.categories.Mark(c.Param("categoryID"), marker, user)
	if err == usecases.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetCategoryEntries returns a list of Entries
// that belong to a Feed that belongs to a Category
func (s *Server) GetCategoryEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	marker := models.MarkerFromString(params.Marker)
	if marker == models.MarkerNone {
		marker = models.MarkerAny
	}

	entries, err := s.categories.Entries(
		c.Param("categoryID"),
		convertOrderByParamToValue(params.OrderBy),
		marker,
		user)
	if err == usecases.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries": entries,
	})
}

// GetCategoryStats returns statistics related to a Category
func (s *Server) GetCategoryStats(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	stats, err := s.categories.Stats(c.Param("categoryID"), user)
	if err == usecases.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stats)
}
