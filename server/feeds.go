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
	"github.com/varddum/syndication/sync"
)

// NewFeed creates a new feed
func (s *Server) NewFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feed := models.Feed{}
	if err := c.Bind(&feed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	entries, err := sync.PullFeed(&feed)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed = userDB.NewFeed(feed.Title, feed.Subscription)

	entries, err = userDB.NewEntries(entries, feed.APIID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error)
	}

	feed.Entries = entries

	return c.JSON(http.StatusCreated, feed)
}

// GetFeeds returns a list of subscribed feeds
func (s *Server) GetFeeds(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feeds := userDB.Feeds()

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	return c.JSON(http.StatusOK, Feeds{
		Feeds: feeds,
	})
}

// GetFeed with id
func (s *Server) GetFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feed, found := userDB.FeedWithAPIID(c.Param("feedID"))
	if !found {
		return c.JSON(http.StatusNotFound, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	return c.JSON(http.StatusOK, feed)
}

// EditFeed with id
func (s *Server) EditFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feed := models.Feed{}

	if err := c.Bind(&feed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed.APIID = c.Param("feedID")

	if _, found := userDB.FeedWithAPIID(feed.APIID); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	err := userDB.EditFeed(&feed)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Feed could not be edited",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// DeleteFeed with id
func (s *Server) DeleteFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feedID := c.Param("feedID")

	if _, found := userDB.FeedWithAPIID(feedID); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}
	err := userDB.DeleteFeed(feedID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Feed could not be deleted",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// MarkFeed applies a Marker to a Feed
func (s *Server) MarkFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feedID := c.Param("feedID")

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.MarkerNone {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	if _, found := userDB.FeedWithAPIID(feedID); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	err := userDB.MarkFeed(feedID, marker)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Feed could not be marked",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetEntriesFromFeed returns a list of entries provided from a feed
func (s *Server) GetEntriesFromFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed, found := userDB.FeedWithAPIID(c.Param("feedID"))
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.MarkerNone {
		markedAs = models.MarkerAny
	}

	entries := userDB.EntriesFromFeed(feed.APIID,
		convertOrderByParamToValue(params.OrderBy),
		markedAs,
	)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	return c.JSON(http.StatusOK, Entries{
		Entries: entries,
	})
}

// GetStatsForFeed provides statistics related to a Feed
func (s *Server) GetStatsForFeed(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	feedID := c.Param("feedID")

	if _, found := userDB.FeedWithAPIID(feedID); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	return c.JSON(http.StatusOK,
		userDB.FeedStats(feedID))
}
