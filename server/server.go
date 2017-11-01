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
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/sync"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"
)

const echoSyndUserKey = "syndUser"

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
		handle        *echo.Echo
		db            *database.DB
		sync          *sync.Sync
		config        config.Server
		versionGroups map[string]*echo.Group
	}

	// ErrorResp represents a common format for error responses returned by a Server
	ErrorResp struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
)

// NewServer creates a new server instance
func NewServer(db *database.DB, sync *sync.Sync, config config.Server) *Server {
	server := Server{
		handle:        echo.New(),
		db:            db,
		sync:          sync,
		config:        config,
		versionGroups: map[string]*echo.Group{},
	}

	server.versionGroups["v1"] = server.handle.Group("v1")

	if config.EnableTLS {
		server.handle.AutoTLSManager.HostPolicy = autocert.HostWhitelist(config.Domain)
		server.handle.AutoTLSManager.Cache = autocert.DirCache(config.CertCacheDir)
	}

	server.registerMiddleware()
	server.registerHandlers()

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

func (s *Server) assumeJSONContentType(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !strings.HasSuffix(c.Path(), "/login") && !strings.HasSuffix(c.Path(), "/register") {
			if c.Request().Header.Get("Content-Type") == "" {
				c.Request().Header.Set("Content-Type", "application/json")
			} else if c.Request().Header.Get("Content-Type") != "application/json" {
				return c.JSON(http.StatusBadRequest, ErrorResp{
					Reason:  "Bad Request",
					Message: "Content should be JSON",
				})
			}
		}

		return next(c)
	}
}

func (s *Server) checkAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == "OPTIONS" {
			return next(c)
		}

		if strings.HasSuffix(c.Path(), "/login") || strings.HasSuffix(c.Path(), "/register") {
			return next(c)
		}

		userClaim := c.Get("user").(*jwt.Token)
		claims := userClaim.Claims.(jwt.MapClaims)
		user, err := s.db.UserWithAPIID(claims["id"].(string))
		if err != nil {
			return c.JSON(http.StatusUnauthorized, ErrorResp{
				Reason:  "Unauthorized",
				Message: "Credentials are invalid",
			})
		}

		key := &models.APIKey{
			Key: userClaim.Raw,
		}
		found, err := s.db.KeyBelongsToUser(key, &user)
		if err != nil || !found {
			return c.JSON(http.StatusUnauthorized, ErrorResp{
				Reason:  "Unauthorized",
				Message: "Credentials are invalid",
			})
		}

		c.Set(echoSyndUserKey, user)

		return next(c)
	}
}

// Stop the server gracefully
func (s *Server) Stop() error {
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

	user, err := s.db.Authenticate(username, password)
	if err != nil {
		// Do not return NotFound errors on invalid credentials
		if dbErr, ok := err.(database.DBError); ok && dbErr.Code() == 404 {
			err = database.Unauthorized{}
		}

		return newError(err, &c)
	}

	key, err := s.db.NewAPIKey(s.config.AuthSecret, &user)
	if err != nil {
		return newError(err, &c)
	}

	return c.JSON(http.StatusOK, key)
}

// Register a user
func (s *Server) Register(c echo.Context) error {
	err := s.db.NewUser(c.FormValue("username"), c.FormValue("password"))
	if err != nil {
		return newError(err, &c)
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

	err := s.db.NewFeed(&feed, &user)
	if err != nil {
		return newError(err, &c)
	}

	err = s.sync.SyncFeed(&feed, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	feed, err := s.db.Feed(c.Param("feedID"), &user)
	if err != nil {
		return newError(err, &c)
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

	err := s.db.EditFeed(&feed, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// DeleteFeed with id
func (s *Server) DeleteFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feedID := c.Param("feedID")
	err := s.db.DeleteFeed(feedID, &user)
	if err != nil {
		return newError(err, &c)
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

	ctg, err := s.db.Category(c.Param("categoryID"), &user)
	if err != nil {
		return newError(err, &c)
	}

	return c.JSON(http.StatusOK, ctg)
}

// GetEntriesFromFeed returns a list of entries provided from a feed
func (s *Server) GetEntriesFromFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return newError(err, &c)
	}

	feed, err := s.db.Feed(c.Param("feedID"), &user)
	if err != nil {
		return newError(err, &c)
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries, err := s.db.EntriesFromFeed(feed.APIID, convertOrderByParamToValue(params.OrderBy), markedAs, &user)
	if err != nil {
		return newError(err, &c)
	}

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
		return newError(err, &c)
	}

	ctg, err := s.db.Category(c.Param("categoryID"), &user)
	if err != nil {
		return newError(err, &c)
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries, err := s.db.EntriesFromCategory(ctg.APIID, convertOrderByParamToValue(params.OrderBy), markedAs, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	feeds, err := s.db.FeedsFromCategory(c.Param("categoryID"), &user)
	if err != nil {
		return newError(err, &c)
	}

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

	err := s.db.NewCategory(&ctg, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	err := s.db.EditCategory(&ctg, &user)
	if err != nil {
		return newError(err, &c)
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
		return newError(err, &c)
	}

	for _, id := range feedIds.Feeds {
		err := s.db.ChangeFeedCategory(id, ctgID, &user)
		if err != nil {
			return newError(err, &c)
		}
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// DeleteCategory with id
func (s *Server) DeleteCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	err := s.db.DeleteCategory(ctgID, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetStatsForCategory returns statistics related to a Category
func (s *Server) GetStatsForCategory(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	ctgID := c.Param("categoryID")

	marks, err := s.db.CategoryStats(ctgID, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	err := s.db.MarkCategory(ctgID, marker, &user)
	if err != nil {
		return newError(err, &c)
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

	err := s.db.NewTag(&tag, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	err := s.db.DeleteTag(tagID, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// EditTag with id
func (s *Server) EditTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag := models.Tag{}
	tag.APIID = c.Param("tagID")

	if err := c.Bind(&tag); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	err := s.db.EditTag(&tag, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetEntriesFromTag returns a list of Entries
// that are tagged by a Tag with ID
func (s *Server) GetEntriesFromTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return newError(err, &c)
	}

	tag, err := s.db.Tag(c.Param("tagID"), &user)
	if err != nil {
		return newError(err, &c)
	}

	withMarker := models.MarkerFromString(params.Marker)
	if withMarker == models.None {
		withMarker = models.Any
	}

	entries, err := s.db.EntriesFromTag(tag.APIID, withMarker, true, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	tag, err := s.db.Tag(c.Param("tagID"), &user)
	if err != nil {
		return newError(err, &c)
	}

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	entryIds := new(EntryIds)
	if err = c.Bind(entryIds); err != nil {
		return newError(err, &c)
	}

	err = s.db.TagEntries(tag.APIID, entryIds.Entries, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetTag with id
func (s *Server) GetTag(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	tag, err := s.db.Tag(c.Param("tagID"), &user)
	if err != nil {
		return newError(err, &c)
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

	err := s.db.MarkFeed(feedID, marker, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetStatsForFeed provides statistics related to a Feed
func (s *Server) GetStatsForFeed(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	feedID := c.Param("feedID")

	marks, err := s.db.FeedStats(feedID, &user)
	if err != nil {
		return newError(err, &c)
	}

	return c.JSON(http.StatusOK, marks)
}

// GetEntry with id
func (s *Server) GetEntry(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	entry, err := s.db.Entry(c.Param("entryID"), &user)
	if err != nil {
		return newError(err, &c)
	}

	return c.JSON(http.StatusOK, entry)
}

// GetEntries returns a list of entries that belong to a user
func (s *Server) GetEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	params := new(EntryQueryParams)
	if err := c.Bind(params); err != nil {
		return newError(err, &c)
	}

	markedAs := models.MarkerFromString(params.Marker)
	if markedAs == models.None {
		markedAs = models.Any
	}

	entries, err := s.db.Entries(convertOrderByParamToValue(params.OrderBy), markedAs, &user)
	if err != nil {
		return newError(err, &c)
	}

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

	err := s.db.MarkEntry(entryID, marker, &user)
	if err != nil {
		return newError(err, &c)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

// GetStatsForEntries provides statistics related to Entries
func (s *Server) GetStatsForEntries(c echo.Context) error {
	user := c.Get(echoSyndUserKey).(models.User)

	return c.JSON(http.StatusOK, s.db.Stats(&user))
}

func (s *Server) registerMiddleware() {
	for version, group := range s.versionGroups {
		group.Use(s.assumeJSONContentType)

		group.Use(middleware.CORS())

		group.Use(middleware.SecureWithConfig(middleware.SecureConfig{
			XSSProtection:      "",
			XFrameOptions:      "",
			ContentTypeNosniff: "nosniff", HSTSMaxAge: 3600,
			ContentSecurityPolicy: "default-src 'self'",
		}))

		group.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
			StackSize:         1 << 10, // 1 KB
			DisablePrintStack: s.config.EnablePanicPrintStack,
		}))

		group.Use(middleware.JWTWithConfig(middleware.JWTConfig{
			Skipper: func(c echo.Context) bool {
				if c.Request().Method == "OPTIONS" {
					return true
				}

				if c.Path() == "/"+version+"/login" || c.Path() == "/"+version+"/register" {
					return true
				}
				return false
			},
			SigningKey:    []byte(s.config.AuthSecret),
			SigningMethod: "HS256",
		}))

		group.Use(s.checkAuth)

		if s.config.EnableRequestLogs {
			group.Use(middleware.Logger())
		}
	}
}

func (s *Server) registerHandlers() {
	v1 := s.versionGroups["v1"]

	v1.POST("/login", s.Login)
	v1.POST("/register", s.Register)

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

func newError(err error, c *echo.Context) error {
	if dbErr, ok := err.(database.DBError); ok {
		return (*c).JSON(dbErr.Code(), ErrorResp{
			Reason:  dbErr.String(),
			Message: dbErr.Error(),
		})
	}

	if syncErr, ok := err.(sync.Error); ok {
		return (*c).JSON(syncErr.Code(), ErrorResp{
			Reason:  syncErr.String(),
			Message: syncErr.Error(),
		})
	}

	return (*c).JSON(http.StatusInternalServerError, ErrorResp{
		Reason:  "InternalServerError",
		Message: "Internal Server Error",
	})
}

func convertOrderByParamToValue(param string) bool {
	if param != "" && strings.ToLower(param) == "oldest" {
		return false
	}

	// Default behavior is to return by newest
	return true
}
