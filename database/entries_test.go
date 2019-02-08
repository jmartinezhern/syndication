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
	"time"

	"github.com/jmartinezhern/syndication/models"
)

func (s *DatabaseTestSuite) TestNewEntry() {
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().NoError(err)
	s.NotZero(feed.ID)
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "John Doe",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err = NewEntry(entry, feed.APIID, s.user)
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
		Author:    "John Doe",
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
		Author:    "John Doe",
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
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().NoError(err)
	s.NotZero(feed.ID)
	s.NotEmpty(feed.APIID)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:     "Test Entry",
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		entries = append(entries, entry)
	}

	entries, err = NewEntries(entries, feed.APIID, s.user)
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
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().NoError(err)

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
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().NoError(err)
	s.NotEmpty(feed.APIID)

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:     "Test Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		entry, err := NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)

		entries = append(entries, entry)
	}

	cEntries, continuationID := Entries(false, models.MarkerUnread, "", 2, s.user)
	s.Equal(entries[2].APIID, continuationID)
	s.Require().Len(cEntries, 2)
	s.Equal(entries[0].Title, cEntries[0].Title)
	s.Equal(entries[1].Title, cEntries[1].Title)

	cEntries, continuationID = Entries(false, models.MarkerUnread, continuationID, 3, s.user)
	s.Len(continuationID, 0)
	s.Require().Len(cEntries, 3)
	s.Equal(entries[2].Title, cEntries[0].Title)
	s.Equal(entries[3].Title, cEntries[1].Title)
	s.Equal(entries[4].Title, cEntries[2].Title)
}

func (s *DatabaseTestSuite) TestEntriesWithNoneMarker() {
	entries, continuationID := Entries(false, models.MarkerNone, "", 1, s.user)
	s.Empty(continuationID)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestEntriesFromFeed() {
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().NoError(err)
	s.NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:     "Test Entry",
		Author:    "John Doe",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err = NewEntry(entry, feed.APIID, s.user)
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
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)

	entry := models.Entry{
		Title:     "Test Entry",
		GUID:      "entry@test",
		Published: time.Now(),
	}

	entry, err = NewEntry(entry, feed.APIID, s.user)
	s.Nil(err)
	s.True(EntryWithGUIDExists(entry.GUID, feed.APIID, s.user))
}

func (s *DatabaseTestSuite) TestEntryWithGUIDDoesNotExists() {
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("Test site", "http://example.com", ctg.APIID, s.user)

	entry := models.Entry{
		Title:     "Test Entry",
		Published: time.Now(),
	}

	_, err = NewEntry(entry, feed.APIID, s.user)
	s.Nil(err)
	s.False(EntryWithGUIDExists("item@test", feed.APIID, s.user))
}

func (s *DatabaseTestSuite) TestMarkEntry() {
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("News", "http://localhost/news", ctg.APIID, s.user)
	s.Require().NoError(err)

	entry := models.Entry{
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err = NewEntry(entry, feed.APIID, s.user)
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

func (s *DatabaseTestSuite) TestMarkAll() {
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("News", "http://localhost/news", ctg.APIID, s.user)

	entry := models.Entry{
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	_, err = NewEntry(entry, feed.APIID, s.user)
	s.Require().NoError(err)

	entries, _ := Entries(true, models.MarkerRead, "", 1, s.user)
	s.Require().Empty(entries)

	MarkAll(models.MarkerRead, s.user)

	entries, _ = Entries(true, models.MarkerRead, "", 1, s.user)
	s.Require().Len(entries, 1)
}

func (s *DatabaseTestSuite) TestDeleteOldEntries() {
	ctg := NewCategory(models.Uncategorized, s.user)
	feed, err := NewFeed("News", "http://localhost/news", ctg.APIID, s.user)

	entry, err := NewEntry(models.Entry{
		Title:     "Article 1",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}, feed.APIID, s.user)
	s.Require().NoError(err)

	savedEntry, err := NewEntry(models.Entry{
		Title:     "Article 2",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
		Saved:     true,
	}, feed.APIID, s.user)
	s.Require().NoError(err)

	s.Require().WithinDuration(time.Now(), entry.CreatedAt, time.Second)

	DeleteOldEntries(entry.CreatedAt.Add(10*time.Second), s.user)

	_, found := EntryWithAPIID(entry.APIID, s.user)
	s.False(found)

	_, found = EntryWithAPIID(savedEntry.APIID, s.user)
	s.True(found)
}
