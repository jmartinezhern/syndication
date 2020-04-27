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

package rest_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	UsersSuite struct {
		suite.Suite

		ctrl      *gomock.Controller
		mockUsers *services.MockUsers

		controller *rest.UsersController
		e          *echo.Echo
	}
)

func (s *UsersSuite) TestDeleteUser() {
	user := models.User{
		ID: utils.CreateID(),
	}

	s.mockUsers.EXPECT().DeleteUser(gomock.Eq(user.ID)).Return(nil)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()

	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, user.ID)
	ctx.SetPath("/v1/users")

	s.NoError(s.controller.DeleteUser(ctx))
	s.Equal(http.StatusNoContent, rec.Code)
}

func (s *UsersSuite) TestDeleteMissingUser() {
	s.mockUsers.EXPECT().DeleteUser(gomock.Any()).Return(services.ErrUserNotFound)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()

	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, "bogus")

	ctx.SetPath("/v1/users")

	s.EqualError(
		s.controller.DeleteUser(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (s *UsersSuite) TestDeleteUserInternalError() {
	s.mockUsers.EXPECT().DeleteUser(gomock.Any()).Return(errors.New("error"))

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()

	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, "bogus")

	ctx.SetPath("/v1/users")

	s.EqualError(
		s.controller.DeleteUser(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (s *UsersSuite) TestGetUser() {
	user := models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}

	s.mockUsers.EXPECT().User(gomock.Eq(user.ID)).Return(user, true)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()

	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, user.ID)
	ctx.SetPath("/v1/users")

	s.NoError(s.controller.GetUser(ctx))
	s.Equal(http.StatusOK, rec.Code)
}

func (s *UsersSuite) TestGetMissingUser() {
	s.mockUsers.EXPECT().User(gomock.Any()).Return(models.User{}, false)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()

	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, "bogus")
	ctx.SetPath("/v1/users")

	s.EqualError(
		s.controller.GetUser(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (s *UsersSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())

	s.e = echo.New()
	s.e.HideBanner = true

	s.mockUsers = services.NewMockUsers(s.ctrl)

	s.controller = rest.NewUsersController(s.mockUsers, s.e)
}

func (s *UsersSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
