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

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/models"
)

type (
	DatabaseTestSuite struct {
		suite.Suite

		db   *DB
		user models.User
	}
)

const TestDatabasePath = "/tmp/syndication-test-db.db"

func (suite *DatabaseTestSuite) SetupTest() {
	var err error
	suite.db, err = NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	suite.Require().NotNil(suite.db)
	suite.Require().Nil(err)

	suite.user = suite.db.NewUser("test", "golang")
	suite.Require().NotZero(suite.user.ID)
}

func (suite *DatabaseTestSuite) TearDownTest() {
	err := suite.db.Close()
	suite.Nil(err)
	err = os.Remove(suite.db.config.Connection)
	suite.Nil(err)
}

func TestNewDB(t *testing.T) {
	_, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	assert.Nil(t, err)
	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestNewDBWithBadOptions(t *testing.T) {
	_, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "bogus",
	})
	assert.NotNil(t, err)
}

func TestNewUser(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUsers(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test_one", "golang")
	db.NewUser("test_two", "password")

	users := db.Users()
	assert.Len(t, users, 2)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUsersWithFields(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test_one", "golang")
	db.NewUser("test_two", "password")

	users := db.Users("uncategorized_category_api_id")
	assert.Len(t, users, 2)
	assert.NotEmpty(t, users[0].UncategorizedCategoryAPIID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDeleteUser(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("first", "golang")
	db.NewUser("second", "password")

	users := db.Users()
	assert.Len(t, users, 2)

	err = db.DeleteUser(users[0].APIID)
	assert.Nil(t, err)

	users = db.Users()
	assert.Len(t, users, 1)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDeleteUnknownUser(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	err = db.DeleteUser("bogus")
	assert.Equal(t, ErrModelNotFound, err)
}

func TestChangeUserName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")
	user, found := db.UserWithName("test")
	require.True(t, found)

	err = db.ChangeUserName(user.APIID, "new_name")
	require.Nil(t, err)

	user, found = db.UserWithName("test")
	assert.False(t, found)
	assert.Zero(t, user.ID)

	user, found = db.UserWithName("new_name")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestChangeUnknownUserName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	err = db.ChangeUserName("bogus", "none")
	assert.Equal(t, ErrModelNotFound, err)
}

func TestChangeUserPassword(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")
	user, found := db.UserWithCredentials("test", "golang")
	require.True(t, found)

	db.ChangeUserPassword(user.APIID, "new_password")

	_, found = db.UserWithCredentials("test", "golang")
	assert.False(t, found)

	user, found = db.UserWithCredentials("test", "new_password")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestSuccessfulAuthentication(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	user := db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	user, found = db.UserWithCredentials("test", "golang")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestBadPasswordAuthentication(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.NotZero(t, user.ID)

	user, found = db.UserWithCredentials("test", "badpass")
	assert.False(t, found)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithAPIID(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	require.True(t, found)

	userWithID, found := db.UserWithAPIID(user.APIID)
	assert.True(t, found)
	assert.Equal(t, user.APIID, userWithID.APIID)
	assert.Equal(t, user.ID, userWithID.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithUnknownAPIID(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	userWithID, found := db.UserWithAPIID("bogus")
	assert.False(t, found)
	assert.Zero(t, userWithID.APIID)
	assert.Zero(t, userWithID.ID)

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("test")
	assert.True(t, found)
	assert.NotZero(t, user.ID)
	assert.NotZero(t, user.APIID)
	assert.Equal(t, user.Username, "test")

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestUserWithUnknownName(t *testing.T) {
	db, err := NewDB(config.Database{
		Connection: TestDatabasePath,
		Type:       "sqlite3",
	})
	require.Nil(t, err)

	db.NewUser("test", "golang")

	user, found := db.UserWithName("bogus")
	assert.False(t, found)
	assert.Zero(t, user.ID)
	assert.Zero(t, user.APIID)
	assert.NotEqual(t, user.Username, "test")

	err = os.Remove(TestDatabasePath)
	assert.Nil(t, err)
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
