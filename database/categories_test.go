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

type CategoriesSuite struct {
	suite.Suite

	user models.User
}

func (s *DatabaseTestSuite) TestNewCategory() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "news",
	}, s.user)

	ctg, found := CategoryWithAPIID(ctgID, s.user)
	s.True(found)
	s.Equal(ctgID, ctg.APIID)
	s.Equal("news", ctg.Name)
}

func (s *DatabaseTestSuite) TestCategories() {
	var ctgs []models.Category
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			APIID: utils.CreateAPIID(),
			Name:  "Test Category " + strconv.Itoa(i),
		}
		CreateCategory(&ctg, s.user)
		ctgs = append(ctgs, ctg)
	}

	cCtgs, continuationID := Categories(s.user, "", 2)
	s.Equal(ctgs[2].APIID, continuationID)
	s.Require().Len(cCtgs, 2)
	s.Equal(ctgs[0].Name, cCtgs[0].Name)
	s.Equal(ctgs[1].Name, cCtgs[1].Name)

	cCtgs, continuationID = Categories(s.user, continuationID, 3)
	s.Len(continuationID, 0)
	s.Require().Len(cCtgs, 3)
	s.Equal(ctgs[2].Name, cCtgs[0].Name)
	s.Equal(ctgs[3].Name, cCtgs[1].Name)
	s.Equal(ctgs[4].Name, cCtgs[2].Name)
}

func (s *DatabaseTestSuite) TestCategoryWithName() {
	CreateCategory(&models.Category{
		Name: "test",
	}, s.user)

	ctg, found := CategoryWithName("test", s.user)
	s.True(found)
	s.Equal("test", ctg.Name)
}

func (s *DatabaseTestSuite) TestEditCategory() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "news",
	}, s.user)

	ctg, err := EditCategory(ctgID, models.Category{Name: "World News"}, s.user)
	s.NoError(err)
	s.Equal("World News", ctg.Name)
}

func (s *DatabaseTestSuite) TestEditNonExistingCategory() {
	_, err := EditCategory("", models.Category{}, s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestDeleteCategory() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "news",
	}, s.user)

	err := DeleteCategory(ctgID, s.user)
	s.NoError(err)

	_, found := CategoryWithAPIID(ctgID, s.user)
	s.False(found)
}

func (s *DatabaseTestSuite) TestDeleteNonExistingCategory() {
	err := DeleteCategory("bogus", s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestCategoryEntries() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "news",
	}, s.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := CreateFeed(&feed, ctgID, s.user)
	s.Require().NoError(err)

	for i := 0; i < 5; i++ {
		var entry models.Entry
		entry = models.Entry{
			Title:     "Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)
	}

	entries := CategoryEntries(ctgID, false, models.MarkerUnread, s.user)
	s.NotEmpty(entries)
	s.Len(entries, 5)
	s.Equal(entries[0].Title, "Entry 0")
}

func (s *DatabaseTestSuite) TestCategoryEntriesWithtNoneMarker() {
	entries := CategoryEntries("bogus", true, models.MarkerNone, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestEntriesFromNonExistingCategory() {
	entries := CategoryEntries("bogus", true, models.MarkerUnread, s.user)
	s.Empty(entries)
}

func (s *DatabaseTestSuite) TestMarkCategory() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "news",
	}, s.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := CreateFeed(&feed, ctgID, s.user)
	s.Require().NoError(err)

	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:     "Test Entry",
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)
	}

	entries := CategoryEntries(ctgID, false, models.MarkerUnread, s.user)
	s.Len(entries, 5)

	err = MarkCategory(ctgID, models.MarkerRead, s.user)
	s.NoError(err)

	entries = CategoryEntries(ctgID, false, models.MarkerRead, s.user)
	s.Len(entries, 5)

	entries = CategoryEntries(ctgID, true, models.MarkerUnread, s.user)
	s.Empty(entries)

	err = MarkCategory(ctgID, models.MarkerUnread, s.user)
	s.NoError(err)

	entries = CategoryEntries(ctgID, false, models.MarkerRead, s.user)
	s.Empty(entries)

	entries = CategoryEntries(ctgID, true, models.MarkerUnread, s.user)
	s.Len(entries, 5)
}

func (s *DatabaseTestSuite) TestMarkUnknownCategory() {
	err := MarkCategory("bogus", models.MarkerRead, s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *DatabaseTestSuite) TestCategoryStats() {
	ctgID := utils.CreateAPIID()
	CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "news",
	}, s.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := CreateFeed(&feed, ctgID, s.user)
	s.Require().NoError(err)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerRead,
			Saved:     true,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		_, err = NewEntry(entry, feed.APIID, s.user)
		s.Require().NoError(err)
	}

	stats := CategoryStats(ctgID, s.user)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(3, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *CategoriesSuite) SetupTest() {
	err := Init("sqlite3", ":memory:")

	s.Require().NoError(err)

	s.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test",
	}
	CreateUser(&s.user)
}

func (s *CategoriesSuite) TearDownTest() {
	err := Close()
	s.NoError(err)
}

func TestCategoriesSuite(t *testing.T) {
	suite.Run(t, new(CategoriesSuite))
}
