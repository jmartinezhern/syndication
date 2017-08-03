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

package models

import (
	"strings"
	"time"
)

// Marker type alias
type Marker int

//Markers
const (
	None = iota
	Read
	Unread
	Any
)

const (
	Uncategorized = "uncategorized"
	Saved         = "saved"
)

func MarkerFromString(marker string) Marker {
	if len(marker) == 0 {
		return None
	}

	value := strings.ToLower(marker)
	if value == "unread" {
		return Unread
	} else if value == "read" {
		return Read
	}

	return None
}

type (
	User struct {
		ID        uint       `json:"-" gorm:"primary_key"`
		CreatedAt time.Time  `json:"created_at,omitempty"`
		UpdatedAt time.Time  `json:"updated_at,omitempty"`
		DeletedAt *time.Time `json:"deleted_at,omitempty" sql:"index"`

		UUID string `json:"id"`

		Categories []Category `json:"categories,omitempty"`
		Feeds      []Feed     `json:"feeds,omitempty"`
		Entries    []Entry    `json:"entries,omitempty"`
		APIKeys    []APIKey   `json:"-"`

		Username                  string `json:"username,required"`
		Email                     string `json:"email,optional"`
		PasswordHash              []byte `json:"-"`
		PasswordSalt              []byte `json:"-"`
		UncategorizedCategoryUUID string `json:"-"`
		SavedCategoryUUID         string `json:"-"`
	}

	Category struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`

		UUID string `json:"id"`

		User   User `json:"-"`
		UserID uint `json:"-"`

		Feeds []Feed `json:"-"`

		Name string `json:"name"`
	}

	Feed struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		UUID string `json:"id"`

		Category   Category
		CategoryID uint `json:"-"`

		User   User `json:"-"`
		UserID uint `json:"-"`

		Entries []Entry `json:"-"`

		Title        string    `json:"title,optional"`
		Description  string    `json:"description,omitempty"`
		Subscription string    `json:"subscription,required"`
		Source       string    `json:"source,omitempty"`
		TTL          int       `json:"ttl,omitempty"`
		Etag         string    `json:"-"`
		LastUpdated  time.Time `json:"-"`
		Status       string    `json:"status,omitempty"`
	}

	Tag struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		UUID string `json:"id"`

		EntryID uint `json:"-"`

		Name string
	}

	Entry struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		UUID string `json:"id"`

		User   User `json:"-"`
		UserID uint `json:"-"`

		Feed   Feed
		FeedID uint `json:"-"`

		Tags []Tag `json:"-"`

		GUID        string    `json:"-"`
		Title       string    `json:"title"`
		Link        string    `json:"link"`
		Description string    `json:"description"`
		Author      string    `json:"author"`
		Published   time.Time `json:"published"`
		Saved       bool      `json:"isSaved"`
		Mark        Marker    `json:"markedAs"`
	}

	Stats struct {
		Unread int `json:"unread"`
		Read   int `json:"read"`
		Saved  int `json:"saved"`
		Total  int `json:"total"`
	}

	APIKey struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"-"`
		UpdatedAt time.Time `json:"-"`

		Key string `json:"token"`

		User   User `json:"-"`
		UserID uint `json:"-"`
	}
)
