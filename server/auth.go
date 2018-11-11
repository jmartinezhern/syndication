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

	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/usecases"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
)

func (s *Server) checkAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if isPathUnauthorized(c) {
			return next(c)
		}

		userClaim := c.Get("user").(*jwt.Token)
		user, isAuth := s.auth.Authenticate(*userClaim)

		if !isAuth {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}

		c.Set(echoSyndUserKey, user)

		return next(c)
	}
}

// Login a user
func (s *Server) Login(c echo.Context) error {
	keys, err := s.auth.Login(c.FormValue("username"), c.FormValue("password"))
	if err == usecases.ErrUserUnauthorized {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, keys)
}

// Register a user
func (s *Server) Register(c echo.Context) error {
	keys, err := s.auth.Register(c.FormValue("username"), c.FormValue("password"))
	if err == usecases.ErrUserConflicts {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, keys)
}

// Renew an API Token
func (s *Server) Renew(c echo.Context) error {
	key := models.APIKeyPair{}
	if err := c.Bind(&key); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	renewedKey, err := s.auth.Renew(key.RefreshKey)
	if err == usecases.ErrUserUnauthorized {
		return echo.NewHTTPError(http.StatusUnauthorized)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, renewedKey)
}
