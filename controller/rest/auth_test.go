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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
)

type (
	AuthControllerSuite struct {
		suite.Suite

		controller *AuthController
		e          *echo.Echo
		db         *sql.DB
	}
)

func (c *AuthControllerSuite) TestRegister() {
	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", "test", "testtesttest"),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.SetPath("/v1/auth/register")

	c.NoError(c.controller.Register(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *AuthControllerSuite) TestLogin() {
	username := "test"
	password := "testtesttest"

	_, err := c.controller.auth.Register(username, password)
	c.NoError(err)

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", username, password),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)

	ctx.SetPath("/v1/auth/login")

	c.NoError(c.controller.Login(ctx))
	c.Equal(http.StatusOK, rec.Code)

	keys := new(models.APIKeyPair)
	c.NoError(json.Unmarshal(rec.Body.Bytes(), keys))
	c.NotEmpty(keys.AccessKey)
	c.NotEmpty(keys.RefreshKey)
}

func (c *AuthControllerSuite) TestRenew() {
	keyPair, err := c.controller.auth.Register("gopher", "testtesttest")
	c.NoError(err)

	req := httptest.NewRequest(
		echo.POST,
		"/",
		strings.NewReader(
			fmt.Sprintf(`{ "refreshToken": "%s" }`, keyPair.RefreshKey),
		),
	)
	req.Header.Add("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)

	ctx.SetPath("/v1/auth/renew")

	c.NoError(c.controller.Renew(ctx))

	key := new(models.APIKey)
	c.NoError(json.Unmarshal(rec.Body.Bytes(), key))
	c.NotEmpty(key.Key)
}

func (c *AuthControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true
	c.e.Logger.SetLevel(log.OFF)
	c.db = sql.NewDB("sqlite3", ":memory:")
	repo := sql.NewUsers(c.db)
	c.controller = NewAuthController(services.NewAuthService("secret", repo), "secret", true, c.e)
}

func (c *AuthControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}
func TestAuthControllerSuite(t *testing.T) {
	suite.Run(t, new(AuthControllerSuite))
}
