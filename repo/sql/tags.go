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
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
)

type (
	Tags struct {
		db *DB
	}
)

func NewTags(db *DB) Tags {
	return Tags{
		db,
	}
}

// Create a new Tag for user
func (t Tags) Create(user *models.User, tag *models.Tag) {
	t.db.db.Model(user).Association("Tags").Append(tag)
}

// TagWithName returns a Tag that has a matching name and belongs to the given user
func (t Tags) TagWithName(user *models.User, name string) (tag models.Tag, found bool) {
	found = !t.db.db.Model(user).Where("name = ?", name).Related(&tag).RecordNotFound()
	return
}

// TagWithID returns a Tag with id that belongs to user
func (t Tags) TagWithID(user *models.User, id string) (tag models.Tag, found bool) {
	found = !t.db.db.Model(user).Where("api_id = ?", id).Related(&tag).RecordNotFound()
	return
}

// List all Tags owned by user
func (t Tags) List(user *models.User, continuationID string, count int) (tags []models.Tag, next string) {
	query := t.db.db.Model(user)

	if continuationID != "" {
		if tag, found := t.TagWithID(user, continuationID); found {
			query = query.Where("id >= ?", tag.ID)
		}
	}

	query.Limit(count + 1).Association("Tags").Find(&tags)

	if len(tags) > count {
		next = tags[len(tags)-1].APIID
		tags = tags[:len(tags)-1]
	}

	return
}

// Update a tag owned by user
func (t Tags) Update(user *models.User, tag *models.Tag) error {
	if dbTag, found := t.TagWithID(user, tag.APIID); found {
		t.db.db.Model(&dbTag).Updates(tag)
		return nil
	}
	return repo.ErrModelNotFound
}

// Delete a tag owned by user
func (t Tags) Delete(user *models.User, id string) error {
	if tag, found := t.TagWithID(user, id); found {
		t.db.db.Delete(&tag)
		return nil
	}
	return repo.ErrModelNotFound
}
