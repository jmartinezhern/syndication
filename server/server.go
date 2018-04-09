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

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"
)

const echoSyndUserDBKey = "syndUserDB"

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
func NewServer(db *database.DB, plugins *plugins.Plugins, config config.Server) *Server {
	server := Server{
		handle:  echo.New(),
		db:      db,
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

		userDB := s.db.NewUserDB(user)

		key := models.APIKey{
			Key: userClaim.Raw,
		}

		found = userDB.KeyBelongsToUser(key)
		if !found {
			return c.JSON(http.StatusUnauthorized, ErrorResp{
				Message: "Credentials are invalid",
			})
		}

		c.Set(echoSyndUserDBKey, userDB)

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

func (s *Server) exportFeeds(exporter importer.FeedImporter, userDB *database.UserDB) ([]byte, error) {
	ctgs := userDB.Categories()

	for idx, ctg := range ctgs {
		ctg.Feeds = userDB.FeedsFromCategory(ctg.APIID)
		ctgs[idx] = ctg
	}

	return exporter.Export(ctgs)
}

func (s *Server) Export(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

	contType := c.Request().Header.Get("Accept")

	var data []byte
	var err error
	switch contType {
	case "application/xml":
		data, err = s.exportFeeds(importer.NewOPMLImporter(), &userDB)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		return c.XMLBlob(http.StatusOK, data)
	}

	return echo.NewHTTPError(http.StatusBadRequest, "Accept header must be set to a supported value")
}

func (s *Server) Import(c echo.Context) error {
	userDB := c.Get(echoSyndUserDBKey).(database.UserDB)

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
		err = s.importFeeds(data, importer.NewOPMLImporter(), &userDB)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return echo.NewHTTPError(http.StatusNoContent)
}

func (s *Server) importFeeds(data []byte, reqImporter importer.FeedImporter, userDB *database.UserDB) error {
	feeds := reqImporter.Import(data)
	for _, feed := range feeds {
		if feed.Category.Name != "" {
			dbCtg := models.Category{}
			if ctg, ok := userDB.CategoryWithName(feed.Category.Name); ok {
				dbCtg = ctg
			} else {
				dbCtg = userDB.NewCategory(feed.Category.Name)
			}

			_, err := userDB.NewFeedWithCategory(feed.Title, feed.Subscription, dbCtg.APIID)
			if err != nil {
				return err
			}
		} else {
			userDB.NewFeed(feed.Title, feed.Subscription)
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
		userDB, ok := c.Get(echoSyndUserDBKey).(database.UserDB)
		if endpoint.NeedsUser && ok {
			userCtx = plugins.NewUserCtx(userDB)
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
