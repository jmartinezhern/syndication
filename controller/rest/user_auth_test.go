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

package rest_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
)

type (
	AuthControllerSuite struct {
		suite.Suite

		ctrl     *gomock.Controller
		mockAuth *services.MockAuth

		controller *rest.AuthController
		e          *echo.Echo
	}
)

func (c *AuthControllerSuite) TestLogin() {
	username := "username"
	password := "password"

	c.mockAuth.EXPECT().Login(gomock.Eq(username), gomock.Eq(password)).Return(models.APIKeyPair{}, nil)

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
}

func (c *AuthControllerSuite) TestLoginUnauthorized() {
	c.mockAuth.EXPECT().Login(gomock.Any(), gomock.Any()).Return(models.APIKeyPair{}, services.ErrUserUnauthorized)

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", "test", "test"),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)

	ctx.SetPath("/v1/auth/login")

	c.EqualError(
		c.controller.Login(ctx),
		echo.NewHTTPError(http.StatusUnauthorized).Error(),
	)
}

func (c *AuthControllerSuite) TestLoginInternalError() {
	c.mockAuth.EXPECT().Login(gomock.Any(), gomock.Any()).Return(models.APIKeyPair{}, errors.New("errors"))

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", "test", "test"),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)

	ctx.SetPath("/v1/auth/login")

	c.EqualError(
		c.controller.Login(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *AuthControllerSuite) TestRegister() {
	username := "test"
	password := "testtesttest"

	c.mockAuth.EXPECT().Register(gomock.Eq(username), gomock.Eq(password)).Return(nil)

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", username, password),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.SetPath("/v1/auth/register")

	c.NoError(c.controller.Register(ctx))
	c.Equal(http.StatusCreated, rec.Code)
}

func (c *AuthControllerSuite) TestRegisterConflict() {
	c.mockAuth.EXPECT().Register(gomock.Any(), gomock.Any()).Return(services.ErrUserConflicts)

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", "test", "testtesttest"),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.SetPath("/v1/auth/register")

	c.EqualError(
		c.controller.Register(ctx),
		echo.NewHTTPError(http.StatusConflict).Error(),
	)
}

func (c *AuthControllerSuite) TestRegisterInternalServer() {
	c.mockAuth.EXPECT().Register(gomock.Any(), gomock.Any()).Return(errors.New("error"))

	req := httptest.NewRequest(
		echo.POST,
		fmt.Sprintf("/?username=%s&password=%s", "test", "testtesttest"),
		nil,
	)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.SetPath("/v1/auth/register")

	c.EqualError(
		c.controller.Register(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *AuthControllerSuite) TestRenew() {
	refreshKey := "key"

	c.mockAuth.EXPECT().Renew(refreshKey).Return(models.APIKey{}, nil)

	req := httptest.NewRequest(
		echo.POST,
		"/",
		strings.NewReader(
			fmt.Sprintf(`{ "refreshToken": "%s" }`, refreshKey),
		),
	)
	req.Header.Add("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)

	ctx.SetPath("/v1/auth/renew")

	c.NoError(c.controller.Renew(ctx))
}

func (c *AuthControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.mockAuth = services.NewMockAuth(c.ctrl)

	c.controller = rest.NewAuthController(c.mockAuth, "secret", true, c.e)
}

func (c *AuthControllerSuite) TearDownTest() {
	c.ctrl.Finish()
}

func TestAuthControllerSuite(t *testing.T) {
	suite.Run(t, new(AuthControllerSuite))
}
