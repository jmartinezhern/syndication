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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type TagsSuite struct {
	suite.Suite

	// TODO: should be interface, not struct
	service  Tag
	tagsRepo repo.Tags
	db       *sql.DB
	user     *models.User
}

func (t *TagsSuite) TestNewTag() {
	tag, err := t.service.New("tech", t.user)
	t.NoError(err)
	t.Equal("tech", tag.Name)
}

func (t *TagsSuite) TestNewConflictingTag() {
	t.tagsRepo.Create(t.user, &models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	})
	_, err := t.service.New("test", t.user)
	t.Equal(ErrTagConflicts, err)
	t.EqualError(err, ErrTagConflicts.Error())
}

func (t *TagsSuite) TestDeleteTag() {
	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.tagsRepo.Create(t.user, &tag)

	err := t.service.Delete(tag.APIID, t.user)
	t.NoError(err)

	_, found := t.tagsRepo.TagWithID(t.user, tag.APIID)
	t.False(found)
}

func (t *TagsSuite) TestDeleteUnknownTag() {
	err := t.service.Delete("bogus", t.user)
	t.Equal(ErrTagNotFound, err)
}

func (t *TagsSuite) TestUpdateTag() {
	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.tagsRepo.Create(t.user, &tag)

	newTag, err := t.service.Update(tag.APIID, "other", t.user)
	t.NoError(err)
	t.Equal("other", newTag.Name)
}

func (t *TagsSuite) TestEditUnknownTag() {
	_, err := t.service.Update("", "", t.user)
	t.EqualError(err, ErrTagNotFound.Error())
}

func (t *TagsSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")

	t.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}
	sql.NewUsers(t.db).Create(t.user)

	t.tagsRepo = sql.NewTags(t.db)

	t.service = NewTagsService(t.tagsRepo, sql.NewEntries(t.db))
}

func (t *TagsSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestTags(t *testing.T) {
	suite.Run(t, new(TagsSuite))
}
