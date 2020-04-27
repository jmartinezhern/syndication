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

package services_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type TagsSuite struct {
	suite.Suite

	service  services.Tags
	tagsRepo repo.Tags
	db       *gorm.DB
	user     *models.User
}

func (t *TagsSuite) TestNewTag() {
	tag, err := t.service.New(t.user.ID, "tech")
	t.NoError(err)
	t.Equal("tech", tag.Name)
}

func (t *TagsSuite) TestNewConflictingTag() {
	t.tagsRepo.Create(t.user.ID, &models.Tag{
		ID:   utils.CreateID(),
		Name: "test",
	})

	_, err := t.service.New(t.user.ID, "test")
	t.EqualError(err, services.ErrTagConflicts.Error())
}

func (t *TagsSuite) TestDeleteTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.tagsRepo.Create(t.user.ID, &tag)

	err := t.service.Delete(t.user.ID, tag.ID)
	t.NoError(err)

	_, found := t.tagsRepo.TagWithID(t.user.ID, tag.ID)
	t.False(found)
}

func (t *TagsSuite) TestDeleteUnknownTag() {
	err := t.service.Delete(t.user.ID, "bogus")
	t.Equal(services.ErrTagNotFound, err)
}

func (t *TagsSuite) TestUpdateTag() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.tagsRepo.Create(t.user.ID, &tag)

	newTag, err := t.service.Update(t.user.ID, tag.ID, "other")
	t.NoError(err)
	t.Equal("other", newTag.Name)
}

func (t *TagsSuite) TestEditUnknownTag() {
	_, err := t.service.Update(t.user.ID, "", "")
	t.EqualError(err, services.ErrTagNotFound.Error())
}

func (t *TagsSuite) SetupTest() {
	var err error

	t.db, err = gorm.Open("sqlite3", ":memory:")
	t.Require().NoError(err)

	sql.AutoMigrateTables(t.db)

	t.user = &models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}
	sql.NewUsers(t.db).Create(t.user)

	t.tagsRepo = sql.NewTags(t.db)

	t.service = services.NewTagsService(t.tagsRepo, sql.NewEntries(t.db))
}

func (t *TagsSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestTags(t *testing.T) {
	suite.Run(t, new(TagsSuite))
}
