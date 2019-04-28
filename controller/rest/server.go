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

// Package controller provides Syndication's REST API.
// See docs/API_reference.md for more information on
// controller requests and responses
package rest

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/acme/autocert"

	"github.com/jmartinezhern/syndication/services"
)

const (
	userContextKey       = "user"
	defaultServerTimeout = time.Second * 5
)

var (
	usersController      *UsersController
	adminAuthController  *AdminAuthController
	authController       *AuthController
	feedsController      *FeedsController
	categoriesController *CategoriesController
	entriesController    *EntriesController
	tagsController       *TagsController
	importerController   *ImporterController
	exporterController   *ExporterController
)

type (

	// Server represents a echo controller instance and holds references to other components
	// needed for the REST API handlers.
	Server struct {
		Timeout time.Duration

		handle *echo.Echo

		isTLSEnabled bool
		port         int
	}

	paginationParams struct {
		ContinuationID string `query:"continuationId"`
		Count          int    `query:"count"`
	}

	listEntriesParams struct {
		ContinuationID string `query:"continuationId"`
		Count          int    `query:"count"`
		Marker         string `query:"markedAs"`
		Saved          bool   `query:"saved"`
		OrderBy        string `query:"orderBy"`
	}

	Controller struct {
		e *echo.Echo
	}
)

// NewServer creates a new controller instance
func NewServer() *Server {
	server := Server{
		handle:  echo.New(),
		Timeout: defaultServerTimeout,
	}

	server.handle.HideBanner = true

	server.registerMiddleware()

	return &server
}

// EnableTLS for the controller instance
func (s *Server) EnableTLS(certCacheDir, domain string) {
	s.handle.AutoTLSManager.HostPolicy = autocert.HostWhitelist(domain)
	s.handle.AutoTLSManager.Cache = autocert.DirCache(certCacheDir)

	s.isTLSEnabled = true
}

// Start the controller
func (s *Server) Start(address string, port int) error {
	s.port = port

	conn := address + ":" + strconv.Itoa(port)
	if s.isTLSEnabled {
		return s.handle.StartAutoTLS(conn)
	}

	return s.handle.Start(conn)
}

// Stop the controller gracefully
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
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

	s.handle.Use(middleware.Logger())
}

func (s *Server) RegisterAdminAuthService(auth services.Auth, secret string) {
	adminAuthController = NewAdminAuthController(auth, secret, s.handle)
}

func (s *Server) RegisterAuthService(auth services.Auth, secret string, allowRegistration bool) {
	//TODO: remove secret param
	authController = NewAuthController(auth, secret, allowRegistration, s.handle)
}

func (s *Server) RegisterUsersService(users services.Users) {
	usersController = NewUsersController(users, s.handle)
}

func (s *Server) RegisterFeedsService(feeds services.Feed) {
	feedsController = NewFeedsController(feeds, s.handle)
}

func (s *Server) RegisterCategoriesService(categories services.Categories) {
	categoriesController = NewCategoriesController(categories, s.handle)
}

func (s *Server) RegisterEntriesService(entries services.Entries) {
	entriesController = NewEntriesController(entries, s.handle)
}

func (s *Server) RegisterTagsController(tags services.Tag) {
	tagsController = NewTagsController(tags, s.handle)
}

func (s *Server) RegisterImporters(importers Importers) {
	importerController = NewCImporterController(importers, s.handle)
}

func (s *Server) RegisterExporters(exporters Exporters) {
	exporterController = NewExporterController(exporters, s.handle)
}

func convertOrderByParamToValue(param string) bool {
	// TODO: change this to be a boolean value
	if param != "" && strings.EqualFold(param, "oldest") {
		return false
	}

	// Default behavior is to return by newest
	return true
}

func getUserID(ctx echo.Context) string {
	token := ctx.Get("token").(*jwt.Token)

	claims := token.Claims.(jwt.MapClaims)

	if claims["type"] != "access" {
		return ""
	}

	return claims["sub"].(string)
}
