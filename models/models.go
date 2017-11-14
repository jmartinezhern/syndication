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

// Markers identify the visibility status of entities
const (
	None = iota
	Read
	Unread
	Any
)

const (
	// Uncategorized identifies an entity as having no category.
	// This only applies to Feeds and Entries
	Uncategorized = "uncategorized"

	// Saved identifies an entity as permenantly saved.
	// This only applies to Entries.
	Saved = "saved"
)

// MarkerFromString converts a string to a Marker type
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
	// User represents a user and owner of all other entities.
	User struct {
		ID        uint       `json:"-" gorm:"primary_key"`
		CreatedAt time.Time  `json:"created_at,omitempty"`
		UpdatedAt time.Time  `json:"updated_at,omitempty"`
		DeletedAt *time.Time `json:"deleted_at,omitempty" sql:"index"`

		APIID string `json:"id"`

		Categories []Category `json:"categories,omitempty"`
		Feeds      []Feed     `json:"feeds,omitempty"`
		Entries    []Entry    `json:"entries,omitempty"`
		APIKeys    []APIKey   `json:"-"`
		Tags       []Tag      `json:"tags,omitempty"`

		Username                   string `json:"username,required"`
		Email                      string `json:"email,optional"`
		PasswordHash               []byte `json:"-"`
		PasswordSalt               []byte `json:"-"`
		UncategorizedCategoryAPIID string `json:"-"`
	}

	// Category represents a container for Feed entities.
	Category struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		UpdatedAt time.Time `json:"updated_at,omitempty"`

		APIID string `json:"id"`

		User   User `json:"-"`
		UserID uint `json:"-"`

		Feeds []Feed `json:"-"`

		Name string `json:"name"`
	}

	// Feed represents an Atom or RSS feed subscription.
	Feed struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		APIID string `json:"id"`

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

	// Tag represents an identifier object that can be applied to Entry objects.
	Tag struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		Name string `json:"name"`

		User   User `json:"-"`
		UserID uint `json:"-"`

		APIID string `json:"id"`

		Entries []Entry `json:"entries" gorm:"many2many:entry_tags;"`
	}

	// Entry represents subscription items obtained from Feed objects.
	Entry struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		APIID string `json:"id"`

		User   User `json:"-"`
		UserID uint `json:"-"`

		Feed   Feed
		FeedID uint `json:"-"`

		Tags []Tag `json:"tags" gorm:"many2many:entry_tags;"`

		GUID      string    `json:"-"`
		Title     string    `json:"title"`
		Link      string    `json:"link"`
		Author    string    `json:"author"`
		Published time.Time `json:"published"`
		Saved     bool      `json:"isSaved"`
		Mark      Marker    `json:"markedAs"`
	}

	// Stats represents statistics related to various attributes of Feed, Entry, and Category objects.
	Stats struct {
		Unread int `json:"unread"`
		Read   int `json:"read"`
		Saved  int `json:"saved"`
		Total  int `json:"total"`
	}

	// APIKey represents an SQL schema for JSON Web Tokens created for User objects.
	APIKey struct {
		ID        uint      `json:"-" gorm:"primary_key"`
		CreatedAt time.Time `json:"-"`
		UpdatedAt time.Time `json:"-"`

		Key string `json:"token"`

		User   User `json:"-"`
		UserID uint `json:"-"`
	}
)
