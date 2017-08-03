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

package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type AuthSuite struct {
	suite.Suite

	usecase Auth
	user    models.User
}

func (t *AuthSuite) TestRegister() {
	keys, err := t.usecase.Register("newUser", "testtesttest")
	t.NoError(err)
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)

	user, found := database.UserWithName("newUser")
	t.True(found)

	t.True(utils.VerifyPasswordHash("testtesttest", user.PasswordHash, user.PasswordSalt))
}

func (t *AuthSuite) TestRegisterConflicting() {
	_, err := t.usecase.Register(t.user.Username, "testtesttest")
	t.EqualError(err, ErrUserConflicts.Error())
}

func (t *AuthSuite) TestLogin() {
	keys, err := t.usecase.Login(t.user.Username, "testtesttest")
	t.NoError(err)
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)
}

func (t *AuthSuite) TestBadLogin() {
	_, err := t.usecase.Login(t.user.Username, "bogus")
	t.Equal(ErrUserUnauthorized, err)
}

func (t *AuthSuite) TestRenew() {
	keys, err := t.usecase.Login(t.user.Username, "testtesttest")
	t.Require().NoError(err)

	time.Sleep(time.Second)

	key, err := t.usecase.Renew(keys.RefreshKey)
	t.NoError(err)
	t.NotEqual(key.Key, keys.AccessKey)
}

func (t *AuthSuite) TestRenewWithInvalidKey() {
	key, err := newAPIKey("secret_cat", models.RefreshKey, t.user)
	t.Require().NoError(err)

	time.Sleep(time.Second)

	_, err = t.usecase.Renew(key.Key)
	t.EqualError(err, ErrUserUnauthorized.Error())
}

func (t *AuthSuite) TestAuthenticate() {
	accessKey, err := newAPIKey("secret_cat", models.AccessKey, t.user)
	t.NoError(err)

	jwtToken, err := jwt.Parse(accessKey.Key, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != signingMethod {
			return nil, errors.New("jwt signing methods mismatch")
		}
		return []byte("secret_cat"), nil
	})
	t.Require().NoError(err)

	_, authed := t.usecase.Authenticate(*jwtToken)
	t.True(authed)
}

func (t *AuthSuite) SetupTest() {
	t.usecase = new(AuthUsecase)

	err := database.Init("sqlite3", ":memory:")
	t.Require().NoError(err)

	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")

	t.user = models.User{
		APIID:        utils.CreateAPIID(),
		Username:     "gopher",
		PasswordHash: hash,
		PasswordSalt: salt,
	}
	database.CreateUser(&t.user)
}

func (t *AuthSuite) TearDownTest() {
	err := database.Close()
	t.NoError(err)
}

func TestAuth(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}
