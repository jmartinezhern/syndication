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
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

type (
	AdminAuthController struct {
		e      *echo.Echo
		auth   services.Auth
		secret string
	}
)

var (
	adminContextKey = "admin"
)

func isUserPath(c echo.Context) bool {
	path := c.Path()
	i := sort.SearchStrings(unauthorizedPaths, path)
	return (i < len(unauthorizedPaths) && unauthorizedPaths[i] == path) || !strings.HasPrefix(path, "/v1/users")
}

func NewAdminAuthController(service services.Auth, secret string, e *echo.Echo) *AdminAuthController {
	e.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper:       isUserPath,
		SigningKey:    []byte(secret),
		SigningMethod: "HS256",
		ContextKey:    "token",
	}))

	v1 := e.Group("v1")

	controller := AdminAuthController{
		e,
		service,
		secret,
	}

	controller.e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if isUserPath(c) {
				return next(c)
			}

			userID := getUserID(c)
			if userID == "" {
				return echo.NewHTTPError(http.StatusUnauthorized)
			}

			c.Set(adminContextKey, userID)

			return next(c)
		}
	})

	v1.POST("/admin/login", controller.Login)
	v1.POST("/admin/renew", controller.Renew)

	return &controller
}

// Login a user
func (s *AdminAuthController) Login(c echo.Context) error {
	keys, err := s.auth.Login(c.FormValue("username"), c.FormValue("password"))
	if err == services.ErrUserUnauthorized {
		return echo.NewHTTPError(http.StatusUnauthorized)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, keys)
}

// Renew an API Token
func (s *AdminAuthController) Renew(c echo.Context) error {
	key := models.APIKeyPair{}
	if err := c.Bind(&key); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	renewedKey, err := s.auth.Renew(key.RefreshKey)
	if err == services.ErrUserUnauthorized {
		return echo.NewHTTPError(http.StatusUnauthorized)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, renewedKey)
}
