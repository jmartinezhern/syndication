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

// Package server provides Syndication's REST API.
// See docs/API_refrence.md for more information on
// server requests and responses
package server

import (
	"bufio"
	"context"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/importer"
	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/plugins"
	"github.com/varddum/syndication/sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"
)

const echoSyndUserKey = "syndUser"

var unauthorizedPaths []string

type (
	// EntryQueryParams maps query parameters used when GETting entries resources
	EntryQueryParams struct {
		Marker  string `query:"markedAs"`
		Saved   bool   `query:"saved"`
		OrderBy string `query:"orderBy"`
	}

	// Server represents a echo server instance and holds references to other components
	// needed for the REST API handlers.
	Server struct {
		handle  *echo.Echo
		db      *database.DB
		sync    *sync.Sync
		plugins *plugins.Plugins
		config  config.Server
		groups  map[string]*echo.Group
	}

	// ErrorResp represents a common format for error responses returned by a Server
	ErrorResp struct {
		Message string `json:"message"`
	}
)

// NewServer creates a new server instance
func NewServer(db *database.DB, sync *sync.Sync, plugins *plugins.Plugins, config config.Server) *Server {
	server := Server{
		handle:  echo.New(),
		db:      db,
		sync:    sync,
		plugins: plugins,
		config:  config,
		groups:  map[string]*echo.Group{},
	}

	server.groups["v1"] = server.handle.Group("v1")
	apiPlugins := plugins.APIPlugins()
	for _, plugin := range apiPlugins {
		for _, endpnt := range plugin.Endpoints() {
			if endpnt.Group != "" {
				server.groups[endpnt.Group] = server.handle.Group(endpnt.Group)
			}
		}
	}

	if config.EnableTLS {
		server.handle.AutoTLSManager.HostPolicy = autocert.HostWhitelist(config.Domain)
		server.handle.AutoTLSManager.Cache = autocert.DirCache(config.CertCacheDir)
	}

	server.registerHandlers()
	server.registerPluginHandlers()
	server.registerMiddleware()

	return &server
}

// Start the server
func (s *Server) Start() error {
	var port string
	if s.config.EnableTLS {
		port = strconv.Itoa(s.config.TLSPort)
	} else {
		port = strconv.Itoa(s.config.HTTPPort)
	}

	var err error
	if s.config.EnableTLS {
		err = s.handle.StartAutoTLS(":" + port)
	} else {
		err = s.handle.Start(":" + port)
	}

	return err
}

func (s *Server) checkAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == "OPTIONS" || isPathUnauthorized(c.Path()) {
			return next(c)
		}

		userClaim := c.Get("user").(*jwt.Token)
		claims := userClaim.Claims.(jwt.MapClaims)
		user, found := s.db.UserWithAPIID(claims["id"].(string))
		if !found {
			return c.JSON(http.StatusUnauthorized, ErrorResp{
				Message: "Credentials are invalid",
			})
		}

		key := models.APIKey{
			Key: userClaim.Raw,
		}

		found = s.db.KeyBelongsToUser(key, &user)
		if !found {
			return c.JSON(http.StatusUnauthorized, ErrorResp{
				Message: "Credentials are invalid",
			})
		}

		c.Set(echoSyndUserKey, user)

		return next(c)
	}
}

// Stop the server gracefully
func (s *Server) Stop() error {
	apiPlugins := s.plugins.APIPlugins()
	for _, plugin := range apiPlugins {
		plugin.Shutdown()
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout.Duration*time.Second)
	defer cancel()
	return s.handle.Shutdown(ctx)
}

// OptionsHandler is a simple default handler for the OPTIONS method.
func (s *Server) OptionsHandler(c echo.Context) error {
	return c.NoContent(200)
}

// Login a user
func (s *Server) Login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	user, found := s.db.UserWithCredentials(username, password)
	if !found {
		return c.JSON(http.StatusUnauthorized, ErrorResp{
			Message: "Credentials are invalid",
		})
	}

	key, err := s.db.NewAPIKey(s.config.AuthSecret, &user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, key)
}

// Register a user
func (s *Server) Register(c echo.Context) error {
	username := c.FormValue("username")
	if _, found := s.db.UserWithName(username); found {
		return c.JSON(http.StatusConflict, ErrorResp{
			Message: "User already exists",
		})
	}

	user := s.db.NewUser(username, c.FormValue("password"))
	if user.ID == 0 {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// NewFeed creates a new feed
func (s *Server) NewFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feed := models.Feed{}
	if err := c.Bind(&feed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed = s.db.NewFeed(feed.Title, feed.Subscription, &user)

	/* TODO
	err := s.sync.SyncFeed(&feed, &user)
	if err != nil {
		return newError(err, &c)
	}*/

	return c.JSON(http.StatusCreated, feed)
}

// GetFeeds returns a list of subscribed feeds
func (s *Server) GetFeeds(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feeds := s.db.Feeds(&user)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	return c.JSON(http.StatusOK, Feeds{
		Feeds: feeds,
	})
}

// GetFeed with id
func (s *Server) GetFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feed, found := s.db.FeedWithAPIID(c.Param("feedID"), &user)
	if !found {
		return c.JSON(http.StatusNotFound, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	return c.JSON(http.StatusOK, feed)
}

// EditFeed with id
func (s *Server) EditFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feed := models.Feed{}

	if err := c.Bind(&feed); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed.APIID = c.Param("feedID")

	if _, found := s.db.FeedWithAPIID(feed.APIID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	err := s.db.EditFeed(&feed, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Feed could not be edited",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// DeleteFeed with id
func (s *Server) DeleteFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feedID := c.Param("feedID")

	if _, found := s.db.FeedWithAPIID(feedID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}
	err := s.db.DeleteFeed(feedID, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Feed could not be deleted",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
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

// GetEntriesFromFeed returns a list of entries provided from a feed
func (s *Server) GetEntriesFromFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	feed, found := s.db.FeedWithAPIID(c.Param("feedID"), &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries := s.db.EntriesFromFeed(feed.APIID,
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

// MarkFeed applies a Marker to a Feed
func (s *Server) MarkFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feedID := c.Param("feedID")

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.None {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	if _, found := s.db.FeedWithAPIID(feedID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	err := s.db.MarkFeed(feedID, marker, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Feed could not be marked",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetStatsForFeed provides statistics related to a Feed
func (s *Server) GetStatsForFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feedID := c.Param("feedID")

	if _, found := s.db.FeedWithAPIID(feedID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Feed does not exist",
		})
	}

	return c.JSON(http.StatusOK,
		s.db.FeedStats(feedID, &user))
}

// GetEntry with id
func (s *Server) GetEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	entryID := c.Param("entryID")

	entry, found := s.db.EntryWithAPIID(entryID, &user)
	if !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Entry does not exist",
		})
	}

	return c.JSON(http.StatusOK, entry)
}

// GetEntries returns a list of entries that belong to a user
func (s *Server) GetEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries := s.db.Entries(convertOrderByParamToValue(params.OrderBy),
		markedAs,
		&user)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	return c.JSON(http.StatusOK, Entries{
		Entries: entries,
	})
}

// MarkEntry applies a Marker to an Entry
func (s *Server) MarkEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	entryID := c.Param("entryID")

	marker := models.MarkerFromString(c.FormValue("as"))
	if marker == models.None {
		return echo.NewHTTPError(http.StatusBadRequest, "'as' parameter is required")
	}

	if _, found := s.db.EntryWithAPIID(entryID, &user); !found {
		return c.JSON(http.StatusBadRequest, ErrorResp{
			Message: "Entry does not exist",
		})
	}

	err := s.db.MarkEntry(entryID, marker, &user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResp{
			Message: "Entry could not be marked",
		})
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetStatsForEntries provides statistics related to Entries
func (s *Server) GetStatsForEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	return c.JSON(http.StatusOK, s.db.Stats(&user))
}

func (s *Server) Export(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	contType := c.Request().Header.Get("Accept")

	var data []byte
	var err error
	switch contType {
	case "application/xml":
		data, err = s.exportFeeds(importer.NewOPMLImporter(), &user)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		return c.XMLBlob(http.StatusOK, data)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "Accept header must be set to a supported value")
}

func (s *Server) exportFeeds(exporter importer.FeedImporter, user *models.User) ([]byte, error) {
	ctgs := s.db.Categories(user)

	for idx, ctg := range ctgs {
		ctg.Feeds = s.db.FeedsFromCategory(ctg.APIID, user)
		ctgs[idx] = ctg
	}

	return exporter.Export(ctgs)
}

func (s *Server) Import(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	contLength := c.Request().ContentLength
	if contLength <= 0 {
		return echo.NewHTTPError(http.StatusNoContent)
	}

	contType := c.Request().Header.Get("Content-Type")
	data := make([]byte, contLength)
	_, err := bufio.NewReader(c.Request().Body).Read(data)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Could not read request body")
	}

	if contType == "" && contLength > 0 {
		contType = http.DetectContentType(data)
	}

	switch contType {
	case "application/xml":
		err = s.importFeeds(data, importer.NewOPMLImporter(), &user)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

func (s *Server) importFeeds(data []byte, reqImporter importer.FeedImporter, user *models.User) error {
	feeds := reqImporter.Import(data)
	for _, feed := range feeds {
		if feed.Category.Name != "" {
			dbCtg := models.Category{}
			if ctg, ok := s.db.CategoryWithName(feed.Category.Name, user); ok {
				dbCtg = ctg
			} else {
				dbCtg = s.db.NewCategory(feed.Category.Name, user)
			}

			_, err := s.db.NewFeedWithCategory(feed.Title, feed.Subscription, dbCtg.APIID, user)
			if err != nil {
				return err
			}
		} else {
			s.db.NewFeed(feed.Title, feed.Subscription, user)
		}
	}

	return nil
}

func (s *Server) registerMiddleware() {
	s.handle.Use(middleware.CORS())

	s.handle.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:      "",
		XFrameOptions:      "",
		ContentTypeNosniff: "nosniff", HSTSMaxAge: 3600,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	s.handle.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         1 << 10, // 1 KB
		DisablePrintStack: s.config.EnablePanicPrintStack,
	}))

	s.handle.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().Method == "OPTIONS" || isPathUnauthorized(c.Path())
		},
		SigningKey:    []byte(s.config.AuthSecret),
		SigningMethod: "HS256",
	}))

	s.handle.Use(s.checkAuth)

	if s.config.EnableRequestLogs {
		s.handle.Use(middleware.Logger())
	}
}

func isPathUnauthorized(path string) bool {
	for _, skpPath := range unauthorizedPaths {
		if skpPath == path {
			return true
		}
	}

	return false
}

func (s *Server) registerHandlers() {
	v1 := s.groups["v1"]

	v1.POST("/login", s.Login)
	v1.POST("/register", s.Register)

	unauthorizedPaths = append(unauthorizedPaths, "/v1/login", "/v1/register")

	v1.POST("/import", s.Import)
	v1.GET("/export", s.Export)

	v1.POST("/feeds", s.NewFeed)
	v1.GET("/feeds", s.GetFeeds)
	v1.GET("/feeds/:feedID", s.GetFeed)
	v1.PUT("/feeds/:feedID", s.EditFeed)
	v1.DELETE("/feeds/:feedID", s.DeleteFeed)
	v1.GET("/feeds/:feedID/entries", s.GetEntriesFromFeed)
	v1.PUT("/feeds/:feedID/mark", s.MarkFeed)
	v1.GET("/feeds/:feedID/stats", s.GetStatsForFeed)
	v1.OPTIONS("/feeds", s.OptionsHandler)
	v1.OPTIONS("/feeds/:feedID", s.OptionsHandler)
	v1.OPTIONS("/feeds/:feedID/mark", s.OptionsHandler)
	v1.OPTIONS("/feeds/:feedID/entries", s.OptionsHandler)
	v1.OPTIONS("/feeds/:feedID/stats", s.OptionsHandler)

	v1.POST("/tags", s.NewTag)
	v1.GET("/tags", s.GetTags)
	v1.GET("/tags/:tagID", s.GetTag)
	v1.DELETE("/tags/:tagID", s.DeleteTag)
	v1.PUT("/tags/:tagID", s.EditTag)
	v1.GET("/tags/:tagID/entries", s.GetEntriesFromTag)
	v1.PUT("/tags/:tagID/entries", s.TagEntries)

	v1.OPTIONS("/tags", s.OptionsHandler)
	v1.OPTIONS("/tags", s.OptionsHandler)
	v1.OPTIONS("/tags/:tagID", s.OptionsHandler)
	v1.OPTIONS("/tags/:tagID", s.OptionsHandler)
	v1.OPTIONS("/tags/:tagID", s.OptionsHandler)
	v1.OPTIONS("/tags/:tagID/entries", s.OptionsHandler)

	v1.POST("/categories", s.NewCategory)
	v1.GET("/categories", s.GetCategories)
	v1.DELETE("/categories/:categoryID", s.DeleteCategory)
	v1.PUT("/categories/:categoryID", s.EditCategory)
	v1.GET("/categories/:categoryID", s.GetCategory)
	v1.PUT("/categories/:categoryID/feeds", s.AddFeedsToCategory)
	v1.GET("/categories/:categoryID/feeds", s.GetFeedsFromCategory)
	v1.GET("/categories/:categoryID/entries", s.GetEntriesFromCategory)
	v1.PUT("/categories/:categoryID/mark", s.MarkCategory)
	v1.GET("/categories/:categoryID/stats", s.GetStatsForCategory)
	v1.OPTIONS("/categories", s.OptionsHandler)
	v1.OPTIONS("/categories/:categoryID", s.OptionsHandler)
	v1.OPTIONS("/categories/:categoryID/mark", s.OptionsHandler)
	v1.OPTIONS("/categories/:categoryID/feeds", s.OptionsHandler)
	v1.OPTIONS("/categories/:categoryID/entries", s.OptionsHandler)
	v1.OPTIONS("/categories/:categoryID/stats", s.OptionsHandler)

	v1.GET("/entries", s.GetEntries)
	v1.GET("/entries/:entryID", s.GetEntry)
	v1.PUT("/entries/:entryID/mark", s.MarkEntry)
	v1.GET("/entries/stats", s.GetStatsForEntries)
	v1.OPTIONS("/entries", s.OptionsHandler)
	v1.OPTIONS("/entries/stats", s.OptionsHandler)
	v1.OPTIONS("/entries/:entryID", s.OptionsHandler)
	v1.OPTIONS("/entries/:entryID/mark", s.OptionsHandler)
}

func (s *Server) registerPluginHandlers() {
	apiPlugins := s.plugins.APIPlugins()
	for _, plugin := range apiPlugins {
		endpoints := plugin.Endpoints()
		for _, endpoint := range endpoints {
			s.registerEndpoint(endpoint)
		}
	}
}

func (s *Server) registerEndpoint(endpoint plugins.Endpoint) {
	var fullPath string
	handlerWrapper := func(c echo.Context) error {
		var ctx plugins.APICtx
		var userCtx plugins.UserCtx
		user, ok := c.Get(echoSyndUserKey).(models.User)
		if endpoint.NeedsUser && ok {
			userCtx = plugins.NewUserCtx(s.db, &user)
			ctx = plugins.APICtx{User: &userCtx}
		} else {
			ctx = plugins.APICtx{}
		}

		endpoint.Handler(ctx, c.Response().Writer, c.Request())
		return nil
	}

	if endpoint.Group != "" {
		grp := s.handle.Group(endpoint.Group)
		s.groups[endpoint.Group] = grp
		grp.Add(endpoint.Method, endpoint.Path, handlerWrapper)

		fullPath = path.Join("/", endpoint.Group, "/", endpoint.Path)
	} else {
		s.handle.Add(endpoint.Method, endpoint.Path, handlerWrapper)

		fullPath = path.Join("/", endpoint.Path)
	}

	if !endpoint.NeedsUser {
		unauthorizedPaths = append(unauthorizedPaths, fullPath)
	}
}

func convertOrderByParamToValue(param string) bool {
	if param != "" && strings.ToLower(param) == "oldest" {
		return false
	}

	// Default behavior is to return by newest
	return true
}
