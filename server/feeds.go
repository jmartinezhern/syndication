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
	"strconv"

	"github.com/labstack/echo"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/usecases"
)

// NewFeed creates a new feed
func (s *Server) NewFeed(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	newFeed := new(models.Feed)
	if err := c.Bind(newFeed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed, err := s.feeds.New(newFeed.Title, newFeed.Subscription, newFeed.Category.APIID, user)
	if err == usecases.ErrFetchingFeed {
		return echo.NewHTTPError(http.StatusBadRequest)
	} else if err == usecases.ErrFeedCategoryNotFound {
		return echo.NewHTTPError(http.StatusBadRequest)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, feed)
}

// GetFeeds returns a list of subscribed feeds
func (s *Server) GetFeeds(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

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

	feeds, next := s.feeds.Feeds(continuationID, count, user)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"feeds":          feeds,
		"continuationID": next,
	})
}

// GetFeed with id
func (s *Server) GetFeed(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	feed, found := s.feeds.Feed(c.Param("feedID"), user)
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, feed)
}

// EditFeed with id
func (s *Server) EditFeed(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	feed := new(models.Feed)
	if err := c.Bind(feed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newFeed, err := s.feeds.Edit(c.Param("feedID"), *feed, user)
	if err == usecases.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, newFeed)
}

// DeleteFeed with id
func (s *Server) DeleteFeed(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	err := s.feeds.Delete(c.Param("feedID"), user)
	if err == usecases.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// MarkFeed applies a Marker to a Feed
func (s *Server) MarkFeed(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.MarkerNone {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	err := s.feeds.Mark(c.Param("feedID"), marker, user)
	if err == usecases.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetFeedEntries returns a list of entries provided from a feed
func (s *Server) GetFeedEntries(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	marker := models.MarkerFromString(params.Marker)
	if marker == models.MarkerNone {
		marker = models.MarkerAny
	}

	entries, err := s.feeds.Entries(
		c.Param("feedID"),
		convertOrderByParamToValue(params.OrderBy),
		marker,
		user,
	)
	if err == usecases.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"entries": entries,
	})
}

// GetFeedStats provides statistics related to a Feed
func (s *Server) GetFeedStats(c echo.Context) error {
	user := c.Get(userContextKey).(models.User)

	stats, err := s.feeds.Stats(c.Param("feedID"), user)
	if err == usecases.ErrFeedNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, stats)
}
