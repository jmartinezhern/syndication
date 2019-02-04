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
	"time"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"

	"github.com/dgrijalva/jwt-go"
)

func (t *UsecasesTestSuite) TestRegister() {
	keys, err := t.auth.Register("newUser", "testtesttest")
	t.NoError(err)
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)

	_, found := database.UserWithCredentials("newUser", "testtesttest")
	t.True(found)
}

func (t *UsecasesTestSuite) TestRegisterConflicting() {
	_, err := t.auth.Register(t.user.Username, "testtesttest")
	t.EqualError(err, ErrUserConflicts.Error())
}

func (t *UsecasesTestSuite) TestLogin() {
	keys, err := t.auth.Login(t.user.Username, "testtesttest")
	t.NoError(err)
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)
}

func (t *UsecasesTestSuite) TestBadLogin() {
	_, err := t.auth.Login(t.user.Username, "bogus")
	t.EqualError(err, ErrUserUnauthorized.Error())
}

func (t *UsecasesTestSuite) TestRenew() {
	keys, err := t.auth.Login(t.user.Username, "testtesttest")
	t.Require().NoError(err)

	time.Sleep(time.Second)

	key, err := t.auth.Renew(keys.RefreshKey)
	t.NoError(err)
	t.NotEqual(key.Key, keys.AccessKey)
}

func (t *UsecasesTestSuite) TestRenewWithInvalidKey() {
	key, err := newAPIKey("secret_cat", models.RefreshKey, t.user)
	t.Require().NoError(err)

	time.Sleep(time.Second)

	_, err = t.auth.Renew(key.Key)
	t.EqualError(err, ErrUserUnauthorized.Error())
}

func (t *UsecasesTestSuite) TestAuthenticate() {
	accessKey, err := newAPIKey("secret_cat", models.AccessKey, t.user)
	t.NoError(err)

	jwtToken, err := jwt.Parse(accessKey.Key, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != signingMethod {
			return nil, errors.New("jwt signing methods mismatch")
		}
		return []byte("secret_cat"), nil
	})
	t.Require().NoError(err)

	_, authed := t.auth.Authenticate(*jwtToken)
	t.True(authed)
}
