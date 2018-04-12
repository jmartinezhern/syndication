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
	"net/http"
	"net/url"
)

func (s *ServerTestSuite) TestRegister() {
	s.gDB.DeleteUser(s.user.APIID)

	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm(testBaseURL+"/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(http.StatusNoContent, regResp.StatusCode)

	users := s.gDB.Users("username")
	s.Len(users, 1)

	s.Equal(randUserName, users[0].Username)
	s.NotEmpty(users[0].APIID)

	err = regResp.Body.Close()
	s.Nil(err)

	s.gDB.DeleteUser(users[0].APIID)
}

func (s *ServerTestSuite) TestRegisterConflictingUser() {
	s.gDB.NewUser("test", "testtesttest")

	regResp, err := http.PostForm(testBaseURL+"/v1/register",
		url.Values{"username": {"test"}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(http.StatusConflict, regResp.StatusCode)
}

func (s *ServerTestSuite) TestLogin() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm(testBaseURL+"/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(http.StatusNoContent, regResp.StatusCode)

	err = regResp.Body.Close()
	s.Require().Nil(err)

	loginResp, err := http.PostForm(testBaseURL+"/v1/login",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(http.StatusOK, loginResp.StatusCode)

	type Token struct {
		Token string `json:"token"`
	}

	var token Token
	err = json.NewDecoder(loginResp.Body).Decode(&token)
	s.Require().Nil(err)
	s.NotEmpty(token.Token)

	user, found := s.gDB.UserWithName(randUserName)
	s.True(found)

	err = loginResp.Body.Close()
	s.Nil(err)

	s.gDB.DeleteUser(user.APIID)
}

func (s *ServerTestSuite) TestLoginWithNonExistentUser() {
	loginResp, err := http.PostForm(testBaseURL+"/v1/login",
		url.Values{"username": {"bogus"}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	s.Equal(http.StatusUnauthorized, loginResp.StatusCode)

	err = loginResp.Body.Close()
	s.Nil(err)
}

func (s *ServerTestSuite) TestLoginWithBadPassword() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm(testBaseURL+"/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	s.Require().Nil(err)

	user, found := s.gDB.UserWithName(randUserName)
	s.Require().True(found)
	defer s.gDB.DeleteUser(user.APIID)

	s.Equal(http.StatusNoContent, regResp.StatusCode)

	err = regResp.Body.Close()
	s.Require().Nil(err)

	loginResp, err := http.PostForm(testBaseURL+"/v1/login",
		url.Values{"username": {randUserName}, "password": {"bogus"}})
	s.Require().Nil(err)

	s.Equal(http.StatusUnauthorized, loginResp.StatusCode)

	err = loginResp.Body.Close()
	s.Nil(err)
}
