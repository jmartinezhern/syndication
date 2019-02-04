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
	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	//"github.com/jmartinezhern/syndication/database"
)

func (t *UsecasesTestSuite) TestNewTag() {
	tag, err := t.tag.New("test", t.user)
	t.NoError(err)
	t.Equal("test", tag.Name)
}

func (t *UsecasesTestSuite) TestNewConflictingTag() {
	database.NewTag("test", t.user)
	_, err := t.tag.New("test", t.user)
	t.EqualError(err, ErrTagConflicts.Error())
}

func (t *UsecasesTestSuite) TestDeleteTag() {
	tag := database.NewTag("test", t.user)
	err := t.tag.Delete(tag.APIID, t.user)
	t.NoError(err)

	_, found := database.TagWithAPIID(tag.APIID, t.user)
	t.False(found)
}

func (t *UsecasesTestSuite) TestDeleteUnknownTag() {
	err := t.tag.Delete("bogus", t.user)
	t.EqualError(err, ErrTagNotFound.Error())
}

func (t *UsecasesTestSuite) TestEditTag() {
	tag := database.NewTag("test", t.user)
	newTag, err := t.tag.Edit(tag.APIID, models.Tag{
		Name: "other",
	}, t.user)
	t.NoError(err)
	t.Equal("other", newTag.Name)
}

func (t *UsecasesTestSuite) TestEditUnknownTag() {
	_, err := t.tag.Edit("bogus", models.Tag{}, t.user)
	t.EqualError(err, ErrTagNotFound.Error())
}
