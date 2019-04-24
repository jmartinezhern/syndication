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

	"github.com/labstack/echo"

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

	v1.POST("/users", controller.CreateUser)
	v1.GET("/users", controller.ListUsers)
	v1.GET("/users/:userID", controller.GetUser)
	v1.DELETE("/users/:userID", controller.DeleteUser)

	return &controller
}

func (c *UsersController) CreateUser(ctx echo.Context) error {
	type NewUser struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var user NewUser
	if err := ctx.Bind(&user); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newUser, err := c.service.NewUser(user.Username, user.Password)
	if err == services.ErrUsernameConflicts {
		return echo.NewHTTPError(http.StatusConflict)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return ctx.JSON(http.StatusCreated, newUser)
}

func (c *UsersController) ListUsers(ctx echo.Context) error {
	params := paginationParams{}
	if err := ctx.Bind(&params); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	users, next := c.service.Users(params.ContinuationID, params.Count)

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"users":          users,
		"continuationID": next,
	})
}

func (c *UsersController) DeleteUser(ctx echo.Context) error {
	userID := ctx.Param("userID")

	err := c.service.DeleteUser(userID)
	if err == services.ErrUserNotFound {
		return echo.NewHTTPError(http.StatusNotFound)
	} else if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (c *UsersController) GetUser(ctx echo.Context) error {
	userID := ctx.Param("userID")

	user, found := c.service.User(userID)
	if !found {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return ctx.JSON(http.StatusOK, user)
}
