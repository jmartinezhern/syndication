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

func (s *DatabaseTestSuite) TestNewCategory() {
	ctg := NewCategory("News", s.user)
	s.NotEmpty(ctg.APIID)
	s.NotZero(ctg.ID)
	s.NotZero(ctg.UserID)
	s.NotZero(ctg.CreatedAt)
	s.NotZero(ctg.UpdatedAt)
	s.NotZero(ctg.UserID)

	query, found := CategoryWithAPIID(ctg.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.Name)
	s.NotZero(query.ID)
	s.NotZero(query.CreatedAt)
	s.NotZero(query.UpdatedAt)
	s.NotZero(query.UserID)
}

func (s *DatabaseTestSuite) TestCategories() {
	for i := 0; i < 5; i++ {
		ctg := NewCategory("Test Category "+strconv.Itoa(i), s.user)
		s.Require().NotZero(ctg.ID)
	}

	ctgs := Categories(s.user)
	s.Len(ctgs, 6)
}

func (s *DatabaseTestSuite) TestCategoryWithName() {
	ctg := NewCategory("Test", s.user)
	dbCtg, found := CategoryWithName("Test", s.user)
	s.True(found)
	s.Equal(ctg.Name, dbCtg.Name)
}

func (s *DatabaseTestSuite) TestEditCategory() {
	ctg := NewCategory("News", s.user)
	s.Require().NotZero(ctg.ID)

	mdfCtg, err := EditCategory(ctg.APIID, models.Category{Name: "World News"}, s.user)
	s.Nil(err)
	s.Equal("World News", mdfCtg.Name)
	s.NotEqual(ctg.Name, mdfCtg.Name)
}

func (s *DatabaseTestSuite) TestEditNonExistingCategory() {
	_, err := EditCategory("", models.Category{}, s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestDeleteCategory() {
	ctg := NewCategory("News", s.user)
	s.Require().NotZero(ctg.ID)

	query, found := CategoryWithAPIID(ctg.APIID, s.user)
	s.True(found)
	s.NotEmpty(query.APIID)

	err := DeleteCategory(ctg.APIID, s.user)
	s.Nil(err)

	_, found = CategoryWithAPIID(ctg.APIID, s.user)
	s.False(found)
}

func (s *DatabaseTestSuite) TestDeleteNonExistingCategory() {
	err := DeleteCategory(createAPIID(), s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestCategoryEntries() {
	firstCtg := NewCategory("News", s.user)
	s.NotEmpty(firstCtg.APIID)

	secondCtg := NewCategory("Tech", s.user)
	s.NotEmpty(secondCtg.APIID)

	firstFeed, err := NewFeedWithCategory("Test site", "http://example.com", firstCtg.APIID, s.user)
	s.Nil(err)

	secondFeed, err := NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID, s.user)
	s.Nil(err)

	thirdFeed, err := NewFeedWithCategory("Test site", "http://example.com", secondCtg.APIID, s.user)
	s.Nil(err)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:     "First Feed Test Entry " + strconv.Itoa(i),
				Author:    "John Doe",
				Link:      "http://example.com",
				Mark:      models.MarkerUnread,
				Published: time.Now(),
			}

			_, err = NewEntry(entry, firstFeed.APIID, s.user)
			s.Require().Nil(err)
		} else if i < 7 {
			entry = models.Entry{
				Title:     "Second Feed Test Entry " + strconv.Itoa(i),
				Author:    "John Doe",
				Link:      "http://example.com",
				Mark:      models.MarkerUnread,
				Published: time.Now(),
			}

			_, err = NewEntry(entry, secondFeed.APIID, s.user)
			s.Require().Nil(err)
		} else {
			entry = models.Entry{
				Title:     "Third Feed Test Entry " + strconv.Itoa(i),
				Author:    "John Doe",
				Link:      "http://example.com",
				Mark:      models.MarkerUnread,
				Published: time.Now(),
			}

			_, err = NewEntry(entry, thirdFeed.APIID, s.user)
			s.Require().Nil(err)
		}

	}

	entries := CategoryEntries(firstCtg.APIID, false, models.MarkerUnread, s.user)
	s.NotEmpty(entries)
	s.Len(entries, 5)
	s.Equal(entries[0].Title, "First Feed Test Entry 0")

	entries = CategoryEntries(secondCtg.APIID, true, models.MarkerUnread, s.user)
	s.NotEmpty(entries)
	s.Len(entries, 5)
	s.Equal(entries[0].Title, "Third Feed Test Entry 9")
	s.Equal(entries[len(entries)-1].Title, "Second Feed Test Entry 5")
}

func (s *DatabaseTestSuite) TestCategoryEntriesWithtNoneMarker() {
	entries := CategoryEntries("bogus", true, models.MarkerNone, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestEntriesFromNonExistingCategory() {
	entries := CategoryEntries(createAPIID(), true, models.MarkerUnread, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestMarkCategory() {
	ctg := NewCategory("News", s.user)

	feed, err := NewFeedWithCategory("Test site", "http://example.com", ctg.APIID, s.user)
	s.Require().Nil(err)

	for i := 0; i < 10; i++ {
		entry := models.Entry{
			Title:     "Test Entry",
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().Nil(err)
	}

	entries := CategoryEntries(ctg.APIID, false, models.MarkerUnread, s.user)
	s.Len(entries, 10)

	err = MarkCategory(ctg.APIID, models.MarkerRead, s.user)
	s.Nil(err)

	entries = CategoryEntries(ctg.APIID, false, models.MarkerRead, s.user)
	s.Len(entries, 10)

	entries = CategoryEntries(ctg.APIID, true, models.MarkerUnread, s.user)
	s.Empty(entries)

	err = MarkCategory(ctg.APIID, models.MarkerUnread, s.user)
	s.Nil(err)

	entries = CategoryEntries(ctg.APIID, false, models.MarkerRead, s.user)
	s.Empty(entries)

	entries = CategoryEntries(ctg.APIID, true, models.MarkerUnread, s.user)
	s.Len(entries, 10)
}

func (s *DatabaseTestSuite) TestMarkUnknownCategory() {
	err := MarkCategory("bogus", models.MarkerRead, s.user)
	s.EqualError(err, ErrModelNotFound.Error())
}

func (s *DatabaseTestSuite) TestCategoryStats() {
	ctg := NewCategory("World", s.user)
	s.Require().NotEmpty(ctg.APIID)

	feed, err := NewFeedWithCategory("News", "http://example.com", ctg.APIID, s.user)
	s.Require().Nil(err)
	s.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerRead,
			Saved:     true,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().Nil(err)
	}

	stats := CategoryStats(ctg.APIID, s.user)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}
