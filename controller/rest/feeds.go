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
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

type (
	FeedsController struct {
		Controller

		feeds services.Feed
	}
)

func NewFeedsController(service services.Feed, e *echo.Echo) *FeedsController {
	v1 := e.Group("v1")

	controller := FeedsController{
		Controller{
			e,
		},
		service,
	}

	v1.POST("/feeds", controller.NewFeed)
	v1.GET("/feeds", controller.GetFeeds)
	v1.GET("/feeds/:feedID", controller.GetFeed)
	v1.PUT("/feeds/:feedID", controller.EditFeed)
	v1.DELETE("/feeds/:feedID", controller.DeleteFeed)
	v1.GET("/feeds/:feedID/entries", controller.GetFeedEntries)
	v1.PUT("/feeds/:feedID/mark", controller.MarkFeed)
	v1.GET("/feeds/:feedID/stats", controller.GetFeedStats)

	return &controller
}

// NewFeed creates a new feed
func (s *FeedsController) NewFeed(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	newFeed := new(models.Feed)
	if err := c.Bind(newFeed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed, err := s.feeds.New(newFeed.Title, newFeed.Subscription, newFeed.Category.ID, userID)
	switch err {
	case services.ErrFetchingFeed:
		return echo.NewHTTPError(http.StatusBadRequest, "subscription url is not reachable")
	case services.ErrFeedCategoryNotFound:
		return echo.NewHTTPError(http.StatusBadRequest, "category does not exist")
	case nil:
		return c.JSON(http.StatusCreated, feed)
	}

	return echo.NewHTTPError(http.StatusInternalServerError)
}

// GetFeeds returns a list of subscribed feeds
func (s *FeedsController) GetFeeds(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	continuationID := c.QueryParam("continuationID")

	count := 100
	countParam := c.QueryParam("count")
	if countParam != "" {
		var err error
		count, err = strconv.Atoi(countParam)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "count must be an integer")
		}
	}

	feeds, next := s.feeds.Feeds(continuationID, count, userID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"feeds":          feeds,
		"continuationID": next,
	})
}

// GetFeed with id
func (s *FeedsController) GetFeed(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	feed, found := s.feeds.Feed(c.Param("feedID"), userID)
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, feed)
}

// EditFeed with id
func (s *FeedsController) EditFeed(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	feed := models.Feed{}
	if err := c.Bind(&feed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed.ID = c.Param("feedID")

	err := s.feeds.Update(&feed, userID)
	if err == services.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, feed)
}

// DeleteFeed with id
func (s *FeedsController) DeleteFeed(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	err := s.feeds.Delete(c.Param("feedID"), userID)
	if err == services.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkFeed applies a Marker to a Feed
func (s *FeedsController) MarkFeed(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	asParam := c.FormValue("as")
	if asParam == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	marker := models.MarkerFromString(asParam)

	err := s.feeds.Mark(c.Param("feedID"), marker, userID)
	if err == services.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetFeedEntries returns a list of entries provided from a feed
func (s *FeedsController) GetFeedEntries(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	params := new(listEntriesParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feedID := c.Param("feedID")

	page := models.Page{
		ContinuationId: params.ContinuationID,
		Count:          params.Count,
		Newest:         convertOrderByParamToValue(params.OrderBy),
		Marker:         models.MarkerFromString(params.Marker),
	}

	entries, next := s.feeds.Entries(feedID, userID, page)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries":        entries,
		"continuationId": next,
	})
}

// GetFeedStats provides statistics related to a Feed
func (s *FeedsController) GetFeedStats(c echo.Context) error {
	userID := c.Get(userContextKey).(string)

	stats, err := s.feeds.Stats(c.Param("feedID"), userID)
	if err == services.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, stats)
}
