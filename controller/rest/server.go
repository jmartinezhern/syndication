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
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

const (
	userContextKey = "user"
)

type (
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

func convertOrderByParamToValue(param string) bool {
	return !(param != "" && strings.EqualFold(param, "oldest"))
}

func getUserID(ctx echo.Context) string {
	token := ctx.Get("token").(*jwt.Token)

	claims := token.Claims.(jwt.MapClaims)

	if claims["type"] != "access" {
		return ""
	}

	return claims["sub"].(string)
}
