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
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type EntriesSuite struct {
	suite.Suite

	ctg  models.Category
	feed models.Feed
	user models.User
}

func (s *EntriesSuite) TestNewEntry() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		Author:    "John Doe",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, s.feed.APIID, s.user)
	s.Require().NoError(err)
	s.NotEmpty(entry.APIID)

	query, found := EntryWithAPIID(entry.APIID, s.user)
	s.True(found)
	s.NotZero(query.FeedID)

	entries := FeedEntries(s.feed.APIID, true, models.MarkerUnread, s.user)
	s.NotEmpty(entries)
	s.Len(entries, 1)
	s.Equal(entries[0].Title, entry.Title)
}

func (s *EntriesSuite) TestNewEntryWithEmptyFeed() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		Link:      "http://example.com",
		Author:    "John Doe",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, utils.CreateAPIID(), s.user)
	s.Error(err)
	s.Equal(ErrModelNotFound, err)
	s.Zero(entry.ID)
	s.Empty(entry.APIID)

	query, found := EntryWithAPIID(entry.APIID, s.user)
	s.False(found)
	s.Zero(query.FeedID)
}

func (s *EntriesSuite) TestNewEntryWithBadFeed() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		Author:    "John Doe",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, utils.CreateAPIID(), s.user)
	s.Error(err)
	s.Zero(entry.ID)
	s.Empty(entry.APIID)

	query, found := EntryWithAPIID(entry.APIID, s.user)
	s.False(found)
	s.Zero(query.FeedID)
}

func (s *EntriesSuite) TestNewEntries() {
	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Test Entry",
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		entries = append(entries, entry)
	}

	entries, err := NewEntries(entries, s.feed.APIID, s.user)
	s.Require().Len(entries, 5)
	s.Require().NoError(err)

	entries = FeedEntries(s.feed.APIID, true, models.MarkerUnread, s.user)
	s.Len(entries, 5)
	for _, entry := range entries {
		s.NotZero(entry.ID)
		s.NotZero(entry.Title)
	}
}

func (s *EntriesSuite) TestNewEntriesWithEmpty() {
	entries, err := NewEntries([]models.Entry{}, s.feed.APIID, s.user)
	s.NoError(err)
	s.Empty(entries)
}

func (s *EntriesSuite) TestNewEntriesWithBadFeed() {
	entries := []models.Entry{
		{},
	}
	_, err := NewEntries(entries, "", s.user)
	s.Error(err)
}

func (s *EntriesSuite) TestEntries() {
	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Test Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		entry, err := NewEntry(entry, s.feed.APIID, s.user)
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

func (s *EntriesSuite) TestEntriesWithNoneMarker() {
	entries, continuationID := Entries(false, models.MarkerNone, "", 1, s.user)
	s.Empty(continuationID)
	s.Empty(entries)
}

func (s *EntriesSuite) TestEntriesFromFeed() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		Author:    "John Doe",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, s.feed.APIID, s.user)
	s.Require().NoError(err)

	entries := FeedEntries(s.feed.APIID, true, models.MarkerUnread, s.user)
	s.Require().NotEmpty(entries)
	s.Equal(entries[0].Title, entry.Title)

	entries = FeedEntries(s.feed.APIID, true, models.MarkerRead, s.user)
	s.Empty(entries)
}

func (s *EntriesSuite) TestEntriesFromFeedWithNoneMarker() {
	entries := FeedEntries("bogus", true, models.MarkerNone, s.user)
	s.Empty(entries)
}

func (s *EntriesSuite) TestEntryWithGUIDExists() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		GUID:      "entry@test",
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, s.feed.APIID, s.user)
	s.NoError(err)
	s.True(EntryWithGUIDExists(entry.GUID, s.feed.APIID, s.user))
}

func (s *EntriesSuite) TestEntryWithGUIDDoesNotExists() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		Published: time.Now(),
	}

	_, err := NewEntry(entry, s.feed.APIID, s.user)
	s.NoError(err)
	s.False(EntryWithGUIDExists("item@test", s.feed.APIID, s.user))
}

func (s *EntriesSuite) TestMarkEntry() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	entry, err := NewEntry(entry, s.feed.APIID, s.user)
	s.Require().NoError(err)

	entries := FeedEntries(s.feed.APIID, true, models.MarkerUnread, s.user)
	s.Require().Len(entries, 1)

	err = MarkEntry(entry.APIID, models.MarkerRead, s.user)
	s.Require().NoError(err)

	entries = FeedEntries(s.feed.APIID, true, models.MarkerUnread, s.user)
	s.Require().Len(entries, 0)

	entries = FeedEntries(s.feed.APIID, true, models.MarkerRead, s.user)
	s.Require().Len(entries, 1)
}

func (s *EntriesSuite) TestMarkUnknownEntry() {
	err := MarkEntry("bogus", models.MarkerRead, s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *EntriesSuite) TestMarkAll() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	_, err := NewEntry(entry, s.feed.APIID, s.user)
	s.Require().NoError(err)

	entries, _ := Entries(true, models.MarkerRead, "", 1, s.user)
	s.Require().Empty(entries)

	MarkAll(models.MarkerRead, s.user)

	entries, _ = Entries(true, models.MarkerRead, "", 1, s.user)
	s.Require().Len(entries, 1)
}

func (s *EntriesSuite) TestDeleteOldEntries() {
	entry, err := NewEntry(models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Article 1",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}, s.feed.APIID, s.user)
	s.Require().NoError(err)

	savedEntry, err := NewEntry(models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Article 2",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
		Saved:     true,
	}, s.feed.APIID, s.user)
	s.Require().NoError(err)

	s.Require().WithinDuration(time.Now(), entry.CreatedAt, time.Second)

	DeleteOldEntries(entry.CreatedAt.Add(10*time.Second), s.user)

	_, found := EntryWithAPIID(entry.APIID, s.user)
	s.False(found)

	_, found = EntryWithAPIID(savedEntry.APIID, s.user)
	s.True(found)
}

func (s *EntriesSuite) SetupTest() {
	err := Init("sqlite3", ":memory:")

	s.Require().NoError(err)

	s.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test",
	}

	CreateUser(&s.user)
	s.ctg = models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	CreateCategory(&s.ctg, s.user)

	s.feed = models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err = CreateFeed(&s.feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)
}

func (s *EntriesSuite) TearDownTest() {
	err := Close()
	s.NoError(err)
}

func TestEntriesSuite(t *testing.T) {
	suite.Run(t, new(EntriesSuite))
}
