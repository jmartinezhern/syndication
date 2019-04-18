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

package sql

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type TagsSuite struct {
	suite.Suite

	user *models.User
	db   *DB
	repo repo.Tags
}

func (s *TagsSuite) TestCreate() {
	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "tech",
	}

	s.repo.Create(s.user, &tag)

	tag, found := s.repo.TagWithID(s.user, tag.APIID)
	s.True(found)
	s.Equal("tech", tag.Name)
}

func (s *TagsSuite) TestDelete() {
	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "news",
	}

	s.repo.Create(s.user, &tag)

	err := s.repo.Delete(s.user, tag.APIID)
	s.NoError(err)

	_, found := s.repo.TagWithID(s.user, tag.APIID)
	s.False(found)
}

func (s *TagsSuite) TestUpdate() {
	tag := models.Tag{
		APIID: utils.CreateAPIID(),
		Name:  "news",
	}

	s.repo.Create(s.user, &tag)

	tag.Name = "World News"

	err := s.repo.Update(s.user, &tag)
	s.NoError(err)

	updatedTag, _ := s.repo.TagWithID(s.user, tag.APIID)
	s.Equal("World News", updatedTag.Name)
}

func (s *TagsSuite) SetupTest() {
	s.db = NewDB("sqlite3", ":memory:")

	s.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test_tags",
	}
	s.db.db.Create(s.user)

	s.repo = NewTags(s.db)
}

func (s *TagsSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestTagsSuite(t *testing.T) {
	suite.Run(t, new(TagsSuite))
}
