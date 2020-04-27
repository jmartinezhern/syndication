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

package sql_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type TagsSuite struct {
	suite.Suite

	user *models.User
	db   *gorm.DB
	repo repo.Tags
}

func (s *TagsSuite) TestCreate() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "tech",
	}

	s.repo.Create(s.user.ID, &tag)

	tag, found := s.repo.TagWithID(s.user.ID, tag.ID)
	s.True(found)
	s.Equal("tech", tag.Name)
}

func (s *TagsSuite) TestDelete() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "news",
	}

	s.repo.Create(s.user.ID, &tag)

	err := s.repo.Delete(s.user.ID, tag.ID)
	s.NoError(err)

	_, found := s.repo.TagWithID(s.user.ID, tag.ID)
	s.False(found)
}

func (s *TagsSuite) TestUpdate() {
	tag := models.Tag{
		ID:   utils.CreateID(),
		Name: "news",
	}

	s.repo.Create(s.user.ID, &tag)

	tag.Name = "World News"

	err := s.repo.Update(s.user.ID, &tag)
	s.NoError(err)

	updatedTag, _ := s.repo.TagWithID(s.user.ID, tag.ID)
	s.Equal("World News", updatedTag.Name)
}

func (s *TagsSuite) SetupTest() {
	var err error

	s.db, err = gorm.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	sql.AutoMigrateTables(s.db)

	s.user = &models.User{
		ID:       utils.CreateID(),
		Username: "test_tags",
	}

	s.db.Create(s.user.ID)

	s.repo = sql.NewTags(s.db)
}

func (s *TagsSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestTagsSuite(t *testing.T) {
	suite.Run(t, new(TagsSuite))
}
