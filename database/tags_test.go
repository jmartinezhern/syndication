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
	"strconv"

	"github.com/varddum/syndication/models"
)

func (s *DatabaseTestSuite) TestNewTag() {
	tag := NewTag("tech", s.user)

	query, found := TagWithAPIID(tag.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.Name)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)
}

func (s *DatabaseTestSuite) TestTagEntries() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		}

		entries = append(entries, entry)
	}

	entries, err := NewEntries(entries, feed.APIID, s.user)
	s.Require().NotEmpty(entries)
	s.Require().Nil(err)

	entries = FeedEntries(feed.APIID, true, models.MarkerAny, s.user)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	tag := NewTag("Tech", s.user)
	s.Nil(err)
	s.NotZero(tag.ID)

	dbTag, found := TagWithAPIID(tag.APIID, s.user)
	s.True(found)
	s.Equal(tag.APIID, dbTag.APIID)
	s.Equal(tag.Name, dbTag.Name)
	s.NotZero(dbTag.ID)

	err = TagEntries(tag.APIID, entryAPIIDs, s.user)
	s.Nil(err)

	taggedEntries := EntriesFromTag(tag.APIID, models.MarkerAny, true, s.user)
	s.Nil(err)
	s.Len(taggedEntries, 5)
}

func (s *DatabaseTestSuite) TestTagUnknownEntries() {
	err := TagEntries("bogus", make([]string, 1), s.user)
	s.EqualError(err, ErrModelNotFound.Error())
}

func (s *DatabaseTestSuite) TestTagNoEntries() {
	err := TagEntries("bogus", nil, s.user)
	s.NoError(err)
}

func (s *DatabaseTestSuite) TestTagMultipleEntries() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry " + strconv.Itoa(i),
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		}

		entries = append(entries, entry)
	}

	_, err := NewEntries(entries, feed.APIID, s.user)
	s.Require().Nil(err)

	secondTag := models.Tag{
		Name: "Second tag",
	}

	firstTag := NewTag("First tag", s.user)
	s.NotZero(firstTag.ID)

	secondTag = NewTag("Second Tag", s.user)
	s.NotZero(secondTag.ID)

	s.Require().NotEqual(firstTag.APIID, secondTag.APIID)

	entries = FeedEntries(feed.APIID, true, models.MarkerAny, s.user)
	s.Require().Nil(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = TagEntries(firstTag.APIID, entryAPIIDs, s.user)
	s.Nil(err)

	err = TagEntries(secondTag.APIID, entryAPIIDs, s.user)
	s.Nil(err)

	taggedEntries := EntriesFromTag(firstTag.APIID, models.MarkerAny, true, s.user)
	s.Len(taggedEntries, 5)

	taggedEntries = EntriesFromTag(secondTag.APIID, models.MarkerAny, true, s.user)
	s.Len(taggedEntries, 5)
}

func (s *DatabaseTestSuite) TestEntriesFromMultipleTags() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	entries := []models.Entry{
		{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		},
		{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		},
	}

	entries, err := NewEntries(entries, feed.APIID, s.user)
	s.Require().Nil(err)

	firstTag := NewTag("First tag", s.user)
	s.NotZero(firstTag.ID)

	secondTag := NewTag("Second tag", s.user)
	s.NotZero(secondTag.ID)

	s.NoError(TagEntries(firstTag.APIID, []string{entries[0].APIID}, s.user))

	s.NoError(TagEntries(secondTag.APIID, []string{entries[1].APIID}, s.user))

	s.Require().NotEqual(firstTag.APIID, secondTag.APIID)

	taggedEntries := EntriesFromMultipleTags([]string{firstTag.APIID, secondTag.APIID}, true, models.MarkerAny, s.user)
	s.Len(taggedEntries, 2)
}

func (s *DatabaseTestSuite) TestDeleteTag() {
	tag := NewTag("News", s.user)

	query, found := TagWithAPIID(tag.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.APIID)

	err := DeleteTag(tag.APIID, s.user)
	s.Nil(err)

	_, found = TagWithAPIID(tag.APIID, s.user)
	s.False(found)
}

func (s *DatabaseTestSuite) TestEditTag() {
	tag := NewTag("News", s.user)

	mdfTag, err := EditTag(tag.APIID, models.Tag{Name: "World News"}, s.user)
	s.Nil(err)
	s.Equal("World News", mdfTag.Name)
	s.NotEqual(tag.Name, mdfTag.Name)
}
