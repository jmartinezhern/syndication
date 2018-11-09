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
// See docs/API_reference.md for more information on
// server requests and responses
package server

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/usecases"
)

const (
	echoSyndUserKey = "syndUser"
)

var unauthorizedPaths []string

var serverShutdownTimeout = time.Second * 5

type (

	// Server represents a echo server instance and holds references to other components
	// needed for the REST API handlers.
	Server struct {
		handle *echo.Echo
		db     *database.DB
		groups map[string]*echo.Group

		eUsecase usecases.Entry
		tUsecase usecases.Tag
		fUsecase usecases.Feed
		cUsecase usecases.Category
		aUsecase usecases.Auth

		isTLSEnabled bool
		port         string

		authSecret string
	}
)

func isPathUnauthorized(c echo.Context) bool {
	path := c.Path()
	i := sort.SearchStrings(unauthorizedPaths, path)
	return i < len(unauthorizedPaths) && unauthorizedPaths[i] == path
}

// NewServer creates a new server instance
func NewServer(authSecret string) *Server {
	server := Server{
		handle:     echo.New(),
		groups:     map[string]*echo.Group{},
		authSecret: authSecret,
		cUsecase:   &usecases.CategoryUsecase{},
		eUsecase:   &usecases.EntryUsecase{},
		fUsecase:   &usecases.FeedUsecase{},
		tUsecase:   &usecases.TagUsecase{},
		aUsecase:   &usecases.AuthUsecase{AuthSecret: authSecret},
	}

	server.groups["v1"] = server.handle.Group("v1")

	server.handle.HideBanner = true

	server.registerHandlers()
	server.registerMiddleware()

	return &server
}

// EnableTLS for the server instance
func (s *Server) EnableTLS(certCacheDir, domain string) {
	s.handle.AutoTLSManager.HostPolicy = autocert.HostWhitelist(domain)
	s.handle.AutoTLSManager.Cache = autocert.DirCache(certCacheDir)

	s.isTLSEnabled = true
}

// Start the server
func (s *Server) Start(address string, port int) error {
	conn := address + ":" + strconv.Itoa(port)
	if s.isTLSEnabled {
		return s.handle.StartAutoTLS(conn)
	}

	return s.handle.Start(conn)
}

// Stop the server gracefully
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	return s.handle.Shutdown(ctx)
}

func (s *Server) registerMiddleware() {
	s.handle.Use(middleware.CORS())

	s.handle.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "",
		XFrameOptions:         "",
		ContentTypeNosniff:    "nosniff",
		HSTSMaxAge:            3600,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	s.handle.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		StackSize:         1 << 10, // 1 KB
		DisablePrintStack: false,
	}))

	s.handle.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper:       isPathUnauthorized,
		SigningKey:    []byte(s.authSecret),
		SigningMethod: "HS256",
	}))

	s.handle.Use(s.checkAuth)
}

func (s *Server) registerHandlers() {
	v1 := s.groups["v1"]

	unauthorizedPaths = append(
		unauthorizedPaths,
		"/v1/auth/login",
		"/v1/auth/register",
		"/v1/auth/renew",
	)

	v1.POST("/auth/login", s.Login)
	v1.POST("/auth/register", s.Register)
	v1.POST("/auth/renew", s.Renew)

	v1.POST("/import", s.Import)
	v1.GET("/export", s.Export)

	v1.POST("/feeds", s.NewFeed)
	v1.GET("/feeds", s.GetFeeds)
	v1.GET("/feeds/:feedID", s.GetFeed)
	v1.PUT("/feeds/:feedID", s.EditFeed)
	v1.DELETE("/feeds/:feedID", s.DeleteFeed)
	v1.GET("/feeds/:feedID/entries", s.GetFeedEntries)
	v1.PUT("/feeds/:feedID/mark", s.MarkFeed)
	v1.GET("/feeds/:feedID/stats", s.GetFeedStats)

	v1.POST("/tags", s.NewTag)
	v1.GET("/tags", s.GetTags)
	v1.GET("/tags/:tagID", s.GetTag)
	v1.DELETE("/tags/:tagID", s.DeleteTag)
	v1.PUT("/tags/:tagID", s.EditTag)
	v1.GET("/tags/:tagID/entries", s.GetEntriesFromTag)
	v1.PUT("/tags/:tagID/entries", s.TagEntries)

	v1.POST("/categories", s.NewCategory)
	v1.GET("/categories", s.GetCategories)
	v1.DELETE("/categories/:categoryID", s.DeleteCategory)
	v1.PUT("/categories/:categoryID", s.EditCategory)
	v1.GET("/categories/:categoryID", s.GetCategory)
	v1.PUT("/categories/:categoryID/feeds", s.AppendCategoryFeeds)
	v1.GET("/categories/:categoryID/feeds", s.GetCategoryFeeds)
	v1.GET("/categories/:categoryID/entries", s.GetCategoryEntries)
	v1.PUT("/categories/:categoryID/mark", s.MarkCategory)
	v1.GET("/categories/:categoryID/stats", s.GetCategoryStats)

	v1.GET("/entries", s.GetEntries)
	v1.GET("/entries/:entryID", s.GetEntry)
	v1.PUT("/entries/:entryID/mark", s.MarkEntry)
	v1.PUT("/entries/mark", s.MarkAllEntries)
	v1.GET("/entries/stats", s.GetEntryStats)
}

func convertOrderByParamToValue(param string) bool {
	if param != "" && strings.ToLower(param) == "oldest" {
		return false
	}

	// Default behavior is to return by newest
	return true
}
