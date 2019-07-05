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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	UsersSuite struct {
		suite.Suite

		e          *echo.Echo
		db         *sql.DB
		usersRepo  repo.Users
		service    services.Users
		controller *UsersController
	}
)

func (s *UsersSuite) TestDeleteUser() {
	user := models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}
	s.usersRepo.Create(&user)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, user.ID)

	ctx.SetPath("/v1/users")

	s.NoError(s.controller.DeleteUser(ctx))
	s.Equal(http.StatusNoContent, rec.Code)
}

func (s *UsersSuite) TestDeleteMissingUser() {
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

func (s *UsersSuite) TestGetUser() {
	user := models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}
	s.usersRepo.Create(&user)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := s.e.NewContext(req, rec)
	ctx.Set(userContextKey, user.ID)

	ctx.SetPath("/v1/users")

	s.NoError(s.controller.GetUser(ctx))

	var respUser models.User
	s.NoError(json.Unmarshal(rec.Body.Bytes(), &respUser))
	s.Equal(user.Username, respUser.Username)
}

func (s *UsersSuite) TestGetMissingUser() {
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
	s.e = echo.New()
	s.e.HideBanner = true

	s.db = sql.NewDB("sqlite3", ":memory:")
	s.usersRepo = sql.NewUsers(s.db)
	s.service = services.NewUsersService(s.usersRepo)
	s.controller = NewUsersController(s.service, s.e)
}

func (s *UsersSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestUsersSuite(t *testing.T) {
	suite.Run(t, new(UsersSuite))
}
