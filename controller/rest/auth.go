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

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

var (
	unauthorizedPaths = []string{
		"/v1/auth/login",
		"/v1/auth/register",
		"/v1/auth/renew",
	}
)

type (
	AuthController struct {
		e      *echo.Echo
		auth   services.Auth
		secret string
	}
)

func isPathUnauthorized(c echo.Context) bool {
	path := c.Path()
	i := sort.SearchStrings(unauthorizedPaths, path)
	return i < len(unauthorizedPaths) && unauthorizedPaths[i] == path
}

func (s *AuthController) checkAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if isPathUnauthorized(c) {
			return next(c)
		}

		userClaim, ok := c.Get("token").(*jwt.Token)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse JWT")
		}
		user, isAuth := s.auth.Authenticate(*userClaim)

		if !isAuth {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}

		c.Set("user", user)

		return next(c)
	}
}

func NewAuthController(service services.Auth, secret string, allowRegistration bool, e *echo.Echo) *AuthController {
	e.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper:       isPathUnauthorized,
		SigningKey:    []byte(secret),
		SigningMethod: "HS256",
		ContextKey:    "token",
	}))

	v1 := e.Group("v1")

	controller := AuthController{
		e,
		service,
		secret,
	}

	controller.e.Use(controller.checkAuth)

	v1.POST("/auth/login", controller.Login)
	if allowRegistration {
		v1.POST("/auth/register", controller.Register)
	}
	v1.POST("/auth/renew", controller.Renew)

	return &controller
}

// Login a user
func (s *AuthController) Login(c echo.Context) error {
	keys, err := s.auth.Login(c.FormValue("username"), c.FormValue("password"))
	if err == services.ErrUserUnauthorized {
		return echo.NewHTTPError(http.StatusUnauthorized)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, keys)
}

// Register a user
func (s *AuthController) Register(c echo.Context) error {
	keys, err := s.auth.Register(c.FormValue("username"), c.FormValue("password"))
	if err == services.ErrUserConflicts {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, keys)
}

// Renew an API Token
func (s *AuthController) Renew(c echo.Context) error {
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
