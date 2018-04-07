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
	"time"

	"github.com/labstack/echo"
)

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

	userDB := s.db.NewUserDB(user)

	key, err := userDB.NewAPIKey(s.config.AuthSecret, time.Hour*72)
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
