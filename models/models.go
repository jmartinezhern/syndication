/*
 *   Copyright (C) 2021. Jorge Martinez Hernandez
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU Affero General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU Affero General Public License for more details.
 *
 *   You should have received a copy of the GNU Affero General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package models

import (
	"encoding/xml"
	"strings"
	"time"
)

// Marker type alias
type Marker = int

// ID type alias
type ID = string

// Markers identify the visibility status of entities
const (
	_ = iota
	MarkerRead
	MarkerUnread
	MarkerAny
)

// APIKeyType alias
type APIKeyType = int

// APIKeyTypes identifies the kind of access or purpose of a key
const (
	RefreshKey APIKeyType = iota
	AccessKey
)

// MarkerFromString converts a string to a Marker type
func MarkerFromString(marker string) Marker {
	value := strings.ToLower(marker)
	switch value {
	case "unread":
		return MarkerUnread
	case "read":
		return MarkerRead
	default:
		return MarkerAny
	}
}

type (
	// User represents a user and owner of all other entities.
	User struct {
		ID        ID         `json:"id" gorm:"primary_key"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		DeletedAt *time.Time `json:"deleted_at" sql:"index"`

		Categories []Category `json:"categories,omitempty"`
		Feeds      []Feed     `json:"feeds,omitempty"`
		Entries    []Entry    `json:"entries,omitempty"`
		APIKeys    []APIKey   `json:"-"`
		Tags       []Tag      `json:"tags,omitempty"`

		Username     string `json:"username"`
		Email        string `json:"email"`
		PasswordHash []byte `json:"-"`
		PasswordSalt []byte `json:"-"`
	}

	// Category represents a container for Feed entities.
	Category struct {
		ID        ID        `json:"id" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		User   User `json:"-"`
		UserID ID   `json:"-"`

		Feeds []Feed `json:"-"`

		Name string `json:"name"`
	}

	// Feed represents an Atom or RSS feed subscription.
	Feed struct {
		ID        ID        `json:"id" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		Category   Category `json:"category,omitempty"`
		CategoryID ID       `json:"-"`

		User   User `json:"-"`
		UserID ID   `json:"-"`

		Entries []Entry `json:"-"`

		Title        string    `json:"title"`
		Description  string    `json:"description,omitempty"`
		Subscription string    `json:"subscription"`
		Source       string    `json:"source,omitempty"`
		TTL          int       `json:"ttl,omitempty"`
		Etag         string    `json:"-"`
		LastUpdated  time.Time `json:"-"`
		Status       string    `json:"status,omitempty"`
	}

	// Tag represents an identifier object that can be applied to Entry objects.
	Tag struct {
		ID        ID        `json:"id" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		Name string `json:"name"`

		User   User `json:"-"`
		UserID ID   `json:"-"`

		Entries []Entry `json:"entries,omitempty" gorm:"many2many:entry_tags;"`
	}

	// Entry represents subscription items obtained from Feed objects.
	Entry struct {
		ID        ID        `json:"id" gorm:"primary_key"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`

		User   User `json:"-"`
		UserID ID   `json:"-"`

		Feed   Feed `json:"-"`
		FeedID ID   `json:"-"`

		Tags []Tag `json:"tags,omitempty" gorm:"many2many:entry_tags;"`

		GUID      string    `json:"-"`
		Title     string    `json:"title"`
		Link      string    `json:"link"`
		Author    string    `json:"author,omitempty"`
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
		ID        ID        `json:"id" gorm:"primary_key"`
		CreatedAt time.Time `json:"-"`
		UpdatedAt time.Time `json:"-"`

		Key  string     `json:"token"`
		Type APIKeyType `json:"-"`

		User    User      `json:"-"`
		UserID  ID        `json:"-"`
		Expires time.Time `json:"expires"`
	}
	// APIKeyPair collects a refresh and access token
	APIKeyPair struct {
		RefreshKey string `json:"refreshToken"`
		AccessKey  string `json:"accessToken"`
	}

	// An OPMLOutline represents an OPML Outline element.
	OPMLOutline struct {
		XMLName xml.Name      `xml:"outline"`
		Type    string        `xml:"type,attr"`
		Text    string        `xml:"text,attr"`
		Title   string        `xml:"title,attr"`
		HTMLUrl string        `xml:"htmlUrl,attr"`
		XMLUrl  string        `xml:"xmlUrl,attr"`
		Items   []OPMLOutline `xml:"outline"`
	}

	// An OPMLBody represents an OPML Body element.
	OPMLBody struct {
		XMLName xml.Name      `xml:"body"`
		Items   []OPMLOutline `xml:"outline"`
	}

	// OPML collects all elements of an OPML file.
	OPML struct {
		XMLName xml.Name `xml:"opml"`
		Body    OPMLBody `xml:"body"`
	}

	Page struct {
		FilterID       string
		ContinuationID string
		Count          int
		Newest         bool
		Marker         Marker
	}
)
