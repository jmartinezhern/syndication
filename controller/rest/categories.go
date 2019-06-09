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
	CategoriesController struct {
		e          *echo.Echo
		categories services.Categories
	}
)

func NewCategoriesController(service services.Categories, e *echo.Echo) *CategoriesController {
	controller := &CategoriesController{
		e,
		service,
	}
	v1 := e.Group("v1")

	v1.POST("/categories", controller.NewCategory)
	v1.GET("/categories", controller.GetCategories)
	v1.DELETE("/categories/:categoryID", controller.DeleteCategory)
	v1.PUT("/categories/:categoryID", controller.EditCategory)
	v1.GET("/categories/:categoryID", controller.GetCategory)
	v1.PUT("/categories/:categoryID/feeds", controller.AppendCategoryFeeds)
	v1.GET("/categories/:categoryID/feeds", controller.GetCategoryFeeds)
	v1.GET("/categories/:categoryID/entries", controller.GetCategoryEntries)
	v1.PUT("/categories/:categoryID/mark", controller.MarkCategory)
	v1.GET("/categories/:categoryID/stats", controller.GetCategoryStats)

	return controller
}

// NewCategory creates a new Categories
func (s *CategoriesController) NewCategory(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	ctg := models.Category{}
	if err := c.Bind(&ctg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newCtg, err := s.categories.New(ctg.Name, userID)
	if err == services.ErrCategoryConflicts {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, newCtg)
}

// GetCategory with id
func (s *CategoriesController) GetCategory(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	ctg, found := s.categories.Category(c.Param("categoryID"), userID)
	if found {
		return c.JSON(http.StatusOK, ctg)
	}

	return echo.NewHTTPError(http.StatusNotFound)
}

// GetCategories returns a list of Categories owned by a user
func (s *CategoriesController) GetCategories(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := paginationParams{}
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	ctgs, next := s.categories.Categories(params.ContinuationID, params.Count, userID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories":     ctgs,
		"continuationID": next,
	})
}

// GetCategoryFeeds returns a list of Feeds that belong to a Categories
func (s *CategoriesController) GetCategoryFeeds(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := paginationParams{}
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}
	ctgID := c.Param("categoryID")
	feeds, next := s.categories.Feeds(ctgID, params.ContinuationID, params.Count, userID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"feeds":          feeds,
		"continuationId": next,
	})
}

// EditCategory with id
func (s *CategoriesController) EditCategory(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	ctg := models.Category{}
	ctgID := c.Param("categoryID")

	if err := c.Bind(&ctg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newCtg, err := s.categories.Update(ctg.Name, ctgID, userID)
	if err == services.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, newCtg)
}

// AppendCategoryFeeds adds a Feed to a Categories with id
func (s *CategoriesController) AppendCategoryFeeds(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	ctgID := c.Param("categoryID")

	type FeedIds struct {
		Feeds []string `json:"feeds"`
	}

	feeds := new(FeedIds)
	if err := c.Bind(feeds); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	s.categories.AddFeeds(ctgID, feeds.Feeds, userID)

	return c.NoContent(http.StatusNoContent)
}

// DeleteCategory with id
func (s *CategoriesController) DeleteCategory(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	err := s.categories.Delete(c.Param("categoryID"), userID)
	if err == services.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkCategory applies a Marker to a Categories
func (s *CategoriesController) MarkCategory(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	marker := models.MarkerFromString(c.FormValue("as"))

	err := s.categories.Mark(c.Param("categoryID"), marker, userID)
	if err == services.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetCategoryEntries returns a list of Entries
// that belong to a Feed that belongs to a Categories
func (s *CategoriesController) GetCategoryEntries(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := listEntriesParams{}
	if err := c.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	ctgID := c.Param("categoryID")

	page := models.Page{
		ContinuationID: params.ContinuationID,
		Count:          params.Count,
		Newest:         convertOrderByParamToValue(params.OrderBy),
		Marker:         models.MarkerFromString(params.Marker),
	}

	entries, next, err := s.categories.Entries(ctgID, userID, page)
	if err == services.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries":        entries,
		"continuationID": next,
	})
}

// GetCategoryStats returns statistics related to a Categories
func (s *CategoriesController) GetCategoryStats(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	stats, err := s.categories.Stats(c.Param("categoryID"), userID)
	if err == services.ErrCategoryNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stats)
}
