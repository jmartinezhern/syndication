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

package database

// This tests take into account the user created in Test Setup

func (s *DatabaseTestSuite) TestNewUser() {
	NewUser("test", "testtesttest")

	user, found := UserWithName("test")
	s.True(found)
	s.NotZero(user.ID)
}

func (s *DatabaseTestSuite) TestUsers() {
	NewUser("test_one", "golang")
	NewUser("test_two", "password")

	users := Users("created_at")
	s.Len(users, 3)
}

func (s *DatabaseTestSuite) TestDeleteUser() {
	NewUser("first", "golang")
	NewUser("second", "password")

	users := Users()
	s.Len(users, 3)

	s.NoError(DeleteUser(users[0].APIID))

	users = Users()
	s.Len(users, 2)

}

func (s *DatabaseTestSuite) TestDeleteUnknownUser() {
	s.EqualError(DeleteUser("bogus"), ErrModelNotFound.Error())
}

func (s *DatabaseTestSuite) TestUserWithAPIID() {
	user := NewUser("test", "testtesttest")

	userWithID, found := UserWithAPIID(user.APIID)
	s.True(found)
	s.Equal(user.ID, userWithID.ID)
	s.Equal(user.APIID, userWithID.APIID)
	s.Equal(user.Username, userWithID.Username)
}

func (s *DatabaseTestSuite) TestUserWithUnknownAPIID() {
	_, found := UserWithAPIID("bogus")
	s.False(found)
}

func (s *DatabaseTestSuite) TestUserWithName() {
	user := NewUser("gopher", "testtesttest")

	userWithName, found := UserWithName("gopher")
	s.True(found)
	s.Equal(user.ID, userWithName.ID)
	s.Equal(user.APIID, userWithName.APIID)
	s.Equal(user.Username, userWithName.Username)
}

func (s *DatabaseTestSuite) TestUserWithUnknownName() {
	_, found := UserWithName("bogus")
	s.False(found)
}

func (s *DatabaseTestSuite) TestChangeUserName() {
	err := ChangeUserName(s.user.APIID, "new_name")
	s.NoError(err)

	_, found := UserWithName("test")
	s.False(found)

	_, found = UserWithName("new_name")
	s.True(found)
}

func (s *DatabaseTestSuite) TestChangeUnknownUserName() {
	err := ChangeUserName("bogus", "new_name")
	s.EqualError(err, ErrModelNotFound.Error())
}

func (s *DatabaseTestSuite) TestChangeUserPassword() {
	err := ChangeUserPassword(s.user.APIID, "new_password")
	s.NoError(err)

	_, found := UserWithCredentials("test", "golang")
	s.False(found)

	_, found = UserWithCredentials("test", "new_password")
	s.True(found)
}

func (s *DatabaseTestSuite) TestChangeUnknownUserPassword() {
	err := ChangeUserPassword("bogus", "gopher")
	s.EqualError(err, ErrModelNotFound.Error())
}
