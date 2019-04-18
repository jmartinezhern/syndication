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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type EntriesSuite struct {
	suite.Suite

	db   *DB
	user *models.User
	repo repo.Entries
}

func (s *EntriesSuite) TestCreate() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		Author:    "John Doe",
		Link:      "http://example.com",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	s.repo.Create(s.user, &entry)

	entry, found := s.repo.EntryWithID(s.user, entry.APIID)
	s.True(found)
	s.Equal("Test Entry", entry.Title)
	s.Equal("John Doe", entry.Author)
}

func (s *EntriesSuite) TestList() {
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Test Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		s.repo.Create(s.user, &entry)
	}

	entries, next := s.repo.List(s.user, "", 2, false, models.MarkerUnread)
	s.Require().Len(entries, 2)
	s.NotEmpty(next)
	s.Equal("Test Entry 0", entries[0].Title)
	s.Equal("Test Entry 1", entries[1].Title)

	entries, _ = s.repo.List(s.user, next, 3, false, models.MarkerUnread)
	s.Require().Len(entries, 3)
	s.Equal(entries[0].APIID, next)
	s.Equal("Test Entry 2", entries[0].Title)
	s.Equal("Test Entry 3", entries[1].Title)
	s.Equal("Test Entry 4", entries[2].Title)
}

func (s *EntriesSuite) TestListFromCategory() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test_category",
	}

	s.db.db.Model(s.user).Association("Categories").Append(&ctg)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	s.db.db.Model(s.user).Association("Feeds").Append(&feed)
	s.db.db.Model(&ctg).Association("Feeds").Append(&feed)

	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		s.db.db.Model(s.user).Association("Entries").Append(&entry)
		s.db.db.Model(&feed).Association("Entries").Append(&entry)
	}

	entries, next := s.repo.ListFromCategory(s.user, ctg.APIID, "", 2, false, models.MarkerUnread)
	s.Require().Len(entries, 2)
	s.NotEmpty(next)
	s.Equal("Entry 0", entries[0].Title)
	s.Equal("Entry 1", entries[1].Title)

	entries, _ = s.repo.ListFromCategory(s.user, ctg.APIID, next, 3, false, models.MarkerUnread)
	s.Require().Len(entries, 3)
	s.Equal(entries[0].APIID, next)
	s.Equal("Entry 2", entries[0].Title)
	s.Equal("Entry 3", entries[1].Title)
	s.Equal("Entry 4", entries[2].Title)
}

func (s *EntriesSuite) TestListFromFeed() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	s.db.db.Model(s.user).Association("Feeds").Append(&feed)

	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		s.db.db.Model(s.user).Association("Entries").Append(&entry)
		s.db.db.Model(&feed).Association("Entries").Append(&entry)
	}

	entries, next := s.repo.ListFromFeed(s.user, feed.APIID, "", 2, false, models.MarkerUnread)
	s.Require().Len(entries, 2)
	s.Equal("Entry 0", entries[0].Title)
	s.Equal("Entry 1", entries[1].Title)

	entries, _ = s.repo.ListFromFeed(s.user, feed.APIID, next, 3, false, models.MarkerUnread)
	s.Require().Len(entries, 3)
	s.Equal(entries[0].APIID, next)
	s.Equal("Entry 2", entries[0].Title)
	s.Equal("Entry 3", entries[1].Title)
	s.Equal("Entry 4", entries[2].Title)
}

func (s *EntriesSuite) TestEntriesWithMissingCategory() {
	entries, _ := s.repo.ListFromCategory(s.user, "bogus", "", 5, true, models.MarkerUnread)
	s.Empty(entries)
}

func (s *EntriesSuite) TestEntryWithGUID() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Test Entry",
		GUID:      "entry@test",
		Published: time.Now(),
	}

	s.repo.Create(s.user, &entry)

	entry, found := s.repo.EntryWithGUID(s.user, entry.GUID)
	s.True(found)
	s.Equal("Test Entry", entry.Title)
	s.Equal("entry@test", entry.GUID)
}

func (s *EntriesSuite) TestMark() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	s.repo.Create(s.user, &entry)

	err := s.repo.Mark(s.user, entry.APIID, models.MarkerRead)
	s.NoError(err)

	entries, _ := s.repo.List(s.user, "", 1, false, models.MarkerRead)
	s.Require().Len(entries, 1)
	s.Equal("Article", entry.Title)
}

func (s *EntriesSuite) TestMarkAll() {
	entry := models.Entry{
		APIID:     utils.CreateAPIID(),
		Title:     "Article",
		Mark:      models.MarkerUnread,
		Published: time.Now(),
	}

	s.repo.Create(s.user, &entry)

	s.repo.MarkAll(s.user, models.MarkerRead)

	entries, _ := s.repo.List(s.user, "", 1, false, models.MarkerRead)
	s.Require().Len(entries, 1)
	s.Equal("Article", entry.Title)
}

func (s *EntriesSuite) TestListFromTags() {
	entries := []models.Entry{
		{
			APIID:  utils.CreateAPIID(),
			Title:  "Test Entry",
			Author: "John Doe",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		},
		{
			APIID:  utils.CreateAPIID(),
			Title:  "Test Entry",
			Author: "John Doe",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		},
	}

	for idx := range entries {
		s.repo.Create(s.user, &entries[idx])
	}

	firstTagID := utils.CreateAPIID()
	s.db.db.Model(s.user).Association("Tags").Append(&models.Tag{
		APIID: firstTagID,
		Name:  "first",
	})

	secondTagID := utils.CreateAPIID()
	s.db.db.Model(s.user).Association("Tags").Append(&models.Tag{
		APIID: secondTagID,
		Name:  "first",
	})

	s.NoError(s.repo.TagEntries(s.user, firstTagID, []string{entries[0].APIID}))
	s.NoError(s.repo.TagEntries(s.user, secondTagID, []string{entries[1].APIID}))

	taggedEntries, _ := s.repo.ListFromTags(s.user, []string{firstTagID, secondTagID}, "", 2, true, models.MarkerAny)
	s.Len(taggedEntries, 2)
}

func (s *EntriesSuite) TestStats() {
	for i := 0; i < 10; i++ {
		var marker models.Marker
		if i < 3 {
			marker = models.MarkerRead
		} else {
			marker = models.MarkerUnread
		}
		entry := models.Entry{
			APIID:     utils.CreateAPIID(),
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      marker,
			Saved:     i < 2,
			Published: time.Now(),
		}

		s.db.db.Model(s.user).Association("Entries").Append(&entry)
	}

	stats := s.repo.Stats(s.user)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(2, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *EntriesSuite) SetupTest() {
	s.db = NewDB("sqlite3", ":memory:")

	s.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test_entries",
	}
	s.db.db.Create(s.user)

	s.repo = NewEntries(s.db)
}

func (s *EntriesSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestEntriesSuite(t *testing.T) {
	suite.Run(t, new(EntriesSuite))
}
