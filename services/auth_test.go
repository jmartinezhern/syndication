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

package services

import (
	"errors"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type AuthSuite struct {
	suite.Suite

	service   Auth
	db        *sql.DB
	usersRepo repo.Users
}

func (t *AuthSuite) TestRegister() {
	keys, err := t.service.Register("newUser", "testtesttest")
	t.NoError(err)
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)

	user, found := t.usersRepo.UserWithName("newUser")
	t.True(found)

	t.True(utils.VerifyPasswordHash("testtesttest", user.PasswordHash, user.PasswordSalt))
}

func (t *AuthSuite) TestRegisterConflicting() {
	t.usersRepo.Create(&models.User{
		APIID:    utils.CreateAPIID(),
		Username: "testUser",
	})
	_, err := t.service.Register("testUser", "testtesttest")
	t.EqualError(err, ErrUserConflicts.Error())
}

func (t *AuthSuite) TestLogin() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")
	t.usersRepo.Create(&models.User{
		APIID:        utils.CreateAPIID(),
		Username:     "testUser",
		PasswordHash: hash,
		PasswordSalt: salt,
	})

	keys, err := t.service.Login("testUser", "testtesttest")
	t.NoError(err)
	t.NotEmpty(keys.AccessKey)
	t.NotEmpty(keys.RefreshKey)
}

func (t *AuthSuite) TestBadLogin() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")
	t.usersRepo.Create(&models.User{
		APIID:        utils.CreateAPIID(),
		Username:     "testUser",
		PasswordHash: hash,
		PasswordSalt: salt,
	})

	_, err := t.service.Login("testUser", "bogus")
	t.Equal(ErrUserUnauthorized, err)
}

func (t *AuthSuite) TestRenew() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")
	t.usersRepo.Create(&models.User{
		APIID:        utils.CreateAPIID(),
		Username:     "testUser",
		PasswordHash: hash,
		PasswordSalt: salt,
	})

	keys, err := t.service.Login("testUser", "testtesttest")
	t.Require().NoError(err)

	time.Sleep(time.Second)

	key, err := t.service.Renew(keys.RefreshKey)
	t.NoError(err)
	t.NotEqual(key.Key, keys.AccessKey)
}

func (t *AuthSuite) TestRenewWithInvalidKey() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")
	user := models.User{
		APIID:        utils.CreateAPIID(),
		Username:     "testUser",
		PasswordHash: hash,
		PasswordSalt: salt,
	}
	t.usersRepo.Create(&user)

	key, err := newAPIKey("secret_cat", models.RefreshKey, &user)
	t.Require().NoError(err)

	time.Sleep(time.Second)

	_, err = t.service.Renew(key.Key)
	t.EqualError(err, ErrUserUnauthorized.Error())
}

func (t *AuthSuite) TestAuthenticate() {
	hash, salt := utils.CreatePasswordHashAndSalt("testtesttest")
	user := models.User{
		APIID:        utils.CreateAPIID(),
		Username:     "testUser",
		PasswordHash: hash,
		PasswordSalt: salt,
	}
	t.usersRepo.Create(&user)

	accessKey, err := newAPIKey("secret_cat", models.AccessKey, &user)
	t.NoError(err)

	jwtToken, err := jwt.Parse(accessKey.Key, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != signingMethod {
			return nil, errors.New("jwt signing methods mismatch")
		}
		return []byte("secret_cat"), nil
	})
	t.Require().NoError(err)

	_, authed := t.service.Authenticate(*jwtToken)
	t.True(authed)
}

func (t *AuthSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")
	t.usersRepo = sql.NewUsers(t.db)
	t.service = NewAuthService("secret", t.usersRepo)
}

func (t *AuthSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestAuth(t *testing.T) {
	suite.Run(t, new(AuthSuite))
}
