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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type TagsSuite struct {
	suite.Suite

	usecase Tag
	tag     models.Tag
	user    models.User
}

func (t *TagsSuite) TestNewTag() {
	tag, err := t.usecase.New("tech", t.user)
	t.NoError(err)
	t.Equal("tech", tag.Name)
}

func (t *TagsSuite) TestNewConflictingTag() {
	_, err := t.usecase.New("test", t.user)
	t.Equal(ErrTagConflicts, err)
	t.EqualError(err, ErrTagConflicts.Error())
}

func (t *TagsSuite) TestDeleteTag() {
	err := t.usecase.Delete(t.tag.APIID, t.user)
	t.NoError(err)

	_, found := database.TagWithAPIID(t.tag.APIID, t.user)
	t.False(found)
}

func (t *TagsSuite) TestDeleteUnknownTag() {
	err := t.usecase.Delete("bogus", t.user)
	t.Equal(ErrTagNotFound, err)
}

func (t *TagsSuite) TestEditTag() {
	newTag, err := t.usecase.Edit(t.tag.APIID, models.Tag{
		Name: "other",
	}, t.user)
	t.NoError(err)
	t.Equal("other", newTag.Name)
}

func (t *TagsSuite) TestEditUnknownTag() {
	_, err := t.usecase.Edit("bogus", models.Tag{}, t.user)
	t.EqualError(err, ErrTagNotFound.Error())
}

func (t *TagsSuite) SetupTest() {
	t.usecase = new(TagUsecase)

	err := database.Init("sqlite3", ":memory:")
	t.Require().NoError(err)

	t.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}
	database.CreateUser(&t.user)

	t.tag = models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	database.CreateTag(&t.tag, t.user)
	t.Require().NoError(err)
}

func (t *TagsSuite) TearDownTest() {
	err := database.Close()
	t.NoError(err)
}

func TestTags(t *testing.T) {
	suite.Run(t, new(TagsSuite))
}
