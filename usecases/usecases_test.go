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
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

type (
	UsecasesTestSuite struct {
		suite.Suite

		user  models.User
		entry Entry
		ctgs  Category
		feed  Feed
		tag   Tag
		auth  Auth
	}
)

const (
	testDBPath = "/tmp/syndication-test-usecases.db"
)

func (t *UsecasesTestSuite) SetupTest() {
	var err error
	t.ctgs = new(CategoryUsecase)
	t.auth = new(AuthUsecase)
	t.entry = new(EntryUsecase)
	t.feed = new(FeedUsecase)
	t.tag = new(TagUsecase)

	err = database.Init("sqlite3", testDBPath)
	t.Require().Nil(err)

	t.user = database.NewUser("gopher", "testtesttest")
}

func (t *UsecasesTestSuite) TearDownTest() {
	os.Remove(testDBPath)
}

func TestUsecasesTestSuite(t *testing.T) {

	suite.Run(t, new(UsecasesTestSuite))
}
