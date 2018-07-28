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
	"github.com/varddum/syndication/models"
)

// NewTag creates a new Tag object owned by user
func (db *DB) NewTag(name string, user models.User) models.Tag {
	tag := models.Tag{}
	if db.db.Model(&user).Where("name = ?", name).Related(&tag).RecordNotFound() {
		tag.Name = name
		tag.APIID = createAPIID()
		db.db.Model(&user).Association("Tags").Append(&tag)
	}

	return tag
}

// NewTag creates a new Tag object owned by user
func NewTag(name string, user models.User) models.Tag {
	return defaultInstance.NewTag(name, user)
}

// Tags returns a list of all Tags owned by user
func (db *DB) Tags(user models.User) (tags []models.Tag) {
	db.db.Model(&user).Association("Tags").Find(&tags)
	return
}

// Tags returns a list of all Tags owned by user
func Tags(user models.User) []models.Tag {
	return defaultInstance.Tags(user)
}

// EditTag for the tag with the given API ID and owned by user
func (db *DB) EditTag(id string, newTag models.Tag, user models.User) (models.Tag, error) {
	if tag, found := db.TagWithAPIID(id, user); found {
		db.db.Model(&tag).Updates(newTag)
		return tag, nil
	}
	return models.Tag{}, ErrModelNotFound
}

// EditTag for the tag with the given API ID and owned by user
func EditTag(id string, newTag models.Tag, user models.User) (models.Tag, error) {
	return defaultInstance.EditTag(id, newTag, user)
}

// DeleteTag with id and owned by user
func (db *DB) DeleteTag(id string, user models.User) error {
	if tag, found := db.TagWithAPIID(id, user); found {
		db.db.Delete(tag)
		return nil
	}
	return ErrModelNotFound
}

// DeleteTag with id and owned by user
func DeleteTag(id string, user models.User) error {
	return defaultInstance.DeleteTag(id, user)
}

// TagWithName returns a Tag that has a matching name and belongs to the given user
func (db *DB) TagWithName(name string, user models.User) (tag models.Tag, found bool) {
	found = !db.db.Model(&user).Where("name = ?", name).Related(&tag).RecordNotFound()
	return
}

// TagWithName returns a Tag that has a matching name and belongs to the given user
func TagWithName(name string, user models.User) (models.Tag, bool) {
	return defaultInstance.TagWithName(name, user)
}

// TagWithAPIID returns a Tag with id that belongs to user
func (db *DB) TagWithAPIID(apiID string, user models.User) (tag models.Tag, found bool) {
	found = !db.db.Model(&user).Where("api_id = ?", apiID).Related(&tag).RecordNotFound()
	return
}

// TagWithAPIID returns a Tag with id that belongs to user
func TagWithAPIID(apiID string, user models.User) (models.Tag, bool) {
	return defaultInstance.TagWithAPIID(apiID, user)
}

func (db *DB) tagPrimaryKey(apiID string, user models.User) uint {
	tag := &models.Tag{}
	if db.db.Model(&user).Where("api_id = ?", apiID).Related(tag).RecordNotFound() {
		return 0
	}
	return tag.ID
}
