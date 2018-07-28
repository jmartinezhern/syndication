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
	"encoding/json"
	"fmt"
	"github.com/varddum/syndication/models"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/varddum/syndication/database"

	"github.com/labstack/echo"
)

func (t *ServerTestSuite) TestRegister() {
	username := "test"
	password := "testtesttest"

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", username, password),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	c := t.e.NewContext(req, t.rec)

	c.SetPath("/v1/auth/register")

	t.NoError(t.server.Register(c))
	t.Equal(http.StatusOK, t.rec.Code)
}

func (t *ServerTestSuite) TestLogin() {
	username := "test"
	password := "testtesttest"

	database.NewUser(username, password)

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", username, password),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	c := t.e.NewContext(req, t.rec)

	c.SetPath("/v1/auth/login")

	t.NoError(t.server.Login(c))
	t.Equal(http.StatusOK, t.rec.Code)

	keys := new(models.APIKeyPair)
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), keys))
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)
}

func (t *ServerTestSuite) TestRenew() {
	keyPair, err := t.server.aUsecase.Register("gopher", "testtesttest")
	t.Require().NoError(err)

	req := httptest.NewRequest(
		echo.POST,
		"/",
		strings.NewReader(
			fmt.Sprintf(`{ "refreshToken": "%s" }`, keyPair.RefreshKey),
		),
	)
	req.Header.Add("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)

	c.SetPath("/v1/auth/renew")

	t.NoError(t.server.Renew(c))
}
