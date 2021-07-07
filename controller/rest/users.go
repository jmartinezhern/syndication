/*
 *   Copyright (C) 2021. Jorge Martinez Hernandez
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU Affero General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU Affero General Public License for more details.
 *
 *   You should have received a copy of the GNU Affero General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/jmartinezhern/syndication/services"
)

type (
	UsersController struct {
		e       *echo.Echo
		service services.Users
	}
)

func NewUsersController(service services.Users, e *echo.Echo) *UsersController {
	v1 := e.Group("v1")

	controller := UsersController{
		e,
		service,
	}

	v1.GET("/users", controller.GetUser)
	v1.DELETE("/users", controller.DeleteUser)

	return &controller
}

func (c *UsersController) DeleteUser(ctx echo.Context) error {
	userID := ctx.Get(userContextKey).(string)

	err := c.service.DeleteUser(userID)
	if err == services.ErrUserNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (c *UsersController) GetUser(ctx echo.Context) error {
	userID := ctx.Get(userContextKey).(string)

	user, found := c.service.User(userID)
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return ctx.JSON(http.StatusOK, user)
}
