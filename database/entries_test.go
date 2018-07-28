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
	"time"

	"github.com/varddum/syndication/models"
)

func (s *DatabaseTestSuite) TestNewEntry() {
	feed := NewFeed("Test site", "http://example.com", s.user)
	s.NotZero(feed.ID)
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, feed.APIID, s.user)
	s.Require().Nil(err)
	s.NotEmpty(entry.APIID)

	query, found := EntryWithAPIID(entry.APIID, s.user)
	s.True(found)
	s.NotZero(query.FeedID)

	entries := FeedEntries(feed.APIID, true, models.MarkerUnread, s.user)
	s.NotEmpty(entries)
	s.Len(entries, 1)
	s.Equal(entries[0].Title, entry.Title)
}

func (s *DatabaseTestSuite) TestNewEntryWithEmptyFeed() {
	entry := models.Entry{
		Title:     "Test Entry",
		Link:      "http://example.com",
		Author:    "varddum",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, createAPIID(), s.user)
	s.NotNil(err)
	s.Equal(ErrModelNotFound, err)
	s.Zero(entry.ID)
	s.Empty(entry.APIID)

	query, found := EntryWithAPIID(entry.APIID, s.user)
	s.False(found)
	s.Zero(query.FeedID)
}

func (s *DatabaseTestSuite) TestNewEntryWithBadFeed() {
	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, createAPIID(), s.user)
	s.NotNil(err)
	s.Zero(entry.ID)
	s.Empty(entry.APIID)

	query, found := EntryWithAPIID(entry.APIID, s.user)
	s.False(found)
	s.Zero(query.FeedID)
}

func (s *DatabaseTestSuite) TestNewEntries() {
	feed := NewFeed("Test site", "http://example.com", s.user)
	s.NotZero(feed.ID)
	s.NotEmpty(feed.APIID)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:     "Test Entry",
			Author:    "varddum",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		entries = append(entries, entry)
	}

	entries, err := NewEntries(entries, feed.APIID, s.user)
	s.Require().Len(entries, 5)
	s.Require().Nil(err)

	entries = FeedEntries(feed.APIID, true, models.MarkerUnread, s.user)
	s.Len(entries, 5)
	for _, entry := range entries {
		s.NotZero(entry.ID)
		s.NotZero(entry.Title)
	}
}

func (s *DatabaseTestSuite) TestNewEntriesWithEmpty() {
	feed := NewFeed("Test site", "http://example.com", s.user)
	entries, err := NewEntries([]models.Entry{}, feed.APIID, s.user)
	s.NoError(err)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestNewEntriesWithBadFeed() {
	entries := []models.Entry{
		{},
	}
	_, err := NewEntries(entries, "", s.user)
	s.NotNil(err)
}

func (s *DatabaseTestSuite) TestEntries() {
	feed := NewFeed("Test site", "http://example.com", s.user)
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, feed.APIID, s.user)
	s.Require().Nil(err)

	entries := Entries(true, models.MarkerUnread, s.user)
	s.NotEmpty(entries)
	s.Equal(entries[0].Title, entry.Title)
}

func (s *DatabaseTestSuite) TestEntriesWithNoneMarker() {
	entries := Entries(false, models.MarkerNone, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestEntriesFromFeed() {
	feed := NewFeed("Test site", "http://example.com", s.user)
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "varddum",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, feed.APIID, s.user)
	s.Require().Nil(err)

	entries := FeedEntries(feed.APIID, true, models.MarkerUnread, s.user)
	s.Require().NotEmpty(entries)
	s.Equal(entries[0].Title, entry.Title)

	entries = FeedEntries(feed.APIID, true, models.MarkerRead, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestEntriesFromFeedWithNoneMarker() {
	entries := FeedEntries("bogus", true, models.MarkerNone, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestEntryWithGUIDExists() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	entry := models.Entry{
		Title:     "Test Entry",
		GUID:      "entry@test",
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, feed.APIID, s.user)
	s.Nil(err)
	s.True(EntryWithGUIDExists(entry.GUID, feed.APIID, s.user))
}

func (s *DatabaseTestSuite) TestEntryWithGUIDDoesNotExists() {
	feed := NewFeed("Test site", "http://example.com", s.user)

	entry := models.Entry{
		Title:     "Test Entry",
		Published: time.Now(),
	}

	_, err := NewEntry(entry, feed.APIID, s.user)
	s.Nil(err)
	s.False(EntryWithGUIDExists("item@test", feed.APIID, s.user))
}

func (s *DatabaseTestSuite) TestMarkEntry() {
	feed := NewFeed("News", "http://localhost/news", s.user)

	entry := models.Entry{
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, feed.APIID, s.user)
	s.Require().Nil(err)

	entries := FeedEntries(feed.APIID, true, models.MarkerUnread, s.user)
	s.Require().Len(entries, 1)

	err = MarkEntry(entry.APIID, models.MarkerRead, s.user)
	s.Require().Nil(err)

	entries = FeedEntries(feed.APIID, true, models.MarkerUnread, s.user)
	s.Require().Len(entries, 0)

	entries = FeedEntries(feed.APIID, true, models.MarkerRead, s.user)
	s.Require().Len(entries, 1)
}

func (s *DatabaseTestSuite) TestMarkUnknownEntry() {
	err := MarkEntry("bogus", models.MarkerRead, s.user)
	s.EqualError(err, ErrModelNotFound.Error())
}
