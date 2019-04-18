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

type CategoriesSuite struct {
	suite.Suite

	repo repo.Categories
	db   *DB
	user *models.User
}

func (s *CategoriesSuite) TestCreate() {
	ctgID := utils.CreateAPIID()

	s.repo.Create(s.user, &models.Category{
		APIID: ctgID,
		Name:  "news",
	})

	ctg, found := s.repo.CategoryWithID(s.user, ctgID)
	s.True(found)
	s.Equal(ctgID, ctg.APIID)
	s.Equal("news", ctg.Name)
}

func (s *CategoriesSuite) TestCategories() {
	var ctgs []models.Category
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			APIID: utils.CreateAPIID(),
			Name:  "Test Category " + strconv.Itoa(i),
		}
		s.repo.Create(s.user, &ctg)
		ctgs = append(ctgs, ctg)
	}

	cCtgs, continuationID := s.repo.List(s.user, "", 2)
	s.Equal(ctgs[2].APIID, continuationID)
	s.Require().Len(cCtgs, 2)
	s.Equal(ctgs[0].Name, cCtgs[0].Name)
	s.Equal(ctgs[1].Name, cCtgs[1].Name)

	cCtgs, continuationID = s.repo.List(s.user, continuationID, 3)
	s.Len(continuationID, 0)
	s.Require().Len(cCtgs, 3)
	s.Equal(ctgs[2].Name, cCtgs[0].Name)
	s.Equal(ctgs[3].Name, cCtgs[1].Name)
	s.Equal(ctgs[4].Name, cCtgs[2].Name)
}

func (s *CategoriesSuite) TestFeeds() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	s.repo.Create(s.user, &ctg)

	for i := 0; i < 5; i++ {
		feed := models.Feed{
			APIID:        utils.CreateAPIID(),
			Title:        "Feed " + strconv.Itoa(i),
			Subscription: "https://example.com",
			Category:     ctg,
		}
		s.db.db.Model(s.user).Association("Feeds").Append(&feed)
	}

	feeds, next := s.repo.Feeds(s.user, ctg.APIID, "", 2)
	s.NotEmpty(next)
	s.Require().Len(feeds, 2)
	s.Equal("Feed 0", feeds[0].Title)
	s.Equal("Feed 1", feeds[1].Title)

	feeds, _ = s.repo.Feeds(s.user, ctg.APIID, next, 3)
	s.Require().Len(feeds, 3)
	s.Equal(feeds[0].APIID, next)
	s.Equal("Feed 2", feeds[0].Title)
	s.Equal("Feed 3", feeds[1].Title)
	s.Equal("Feed 4", feeds[2].Title)
}

func (s *CategoriesSuite) TestUncategorized() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			APIID:        utils.CreateAPIID(),
			Title:        "Feed " + strconv.Itoa(i),
			Subscription: "https://example.com",
		}
		s.db.db.Model(s.user).Association("Feeds").Append(&feed)
	}

	feeds, next := s.repo.Uncategorized(s.user, "", 2)
	s.NotEmpty(next)
	s.Require().Len(feeds, 2)
	s.Equal("Feed 0", feeds[0].Title)
	s.Equal("Feed 1", feeds[1].Title)

	feeds, _ = s.repo.Uncategorized(s.user, next, 3)
	s.Require().Len(feeds, 3)
	s.Equal(feeds[0].APIID, next)
	s.Equal("Feed 2", feeds[0].Title)
	s.Equal("Feed 3", feeds[1].Title)
	s.Equal("Feed 4", feeds[2].Title)
}

func (s *CategoriesSuite) TestCategoryWithName() {
	s.repo.Create(s.user, &models.Category{
		Name: "test",
	})

	ctg, found := s.repo.CategoryWithName(s.user, "test")
	s.True(found)
	s.Equal("test", ctg.Name)
}

func (s *CategoriesSuite) TestUpdate() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "news",
	}
	s.repo.Create(s.user, &ctg)

	ctg.Name = "updates"

	err := s.repo.Update(s.user, &ctg)
	s.NoError(err)

	updatedCtg, _ := s.repo.CategoryWithID(s.user, ctg.APIID)
	s.Equal(ctg.Name, updatedCtg.Name)
}

func (s *CategoriesSuite) TestUpdateMissing() {
	err := s.repo.Update(s.user, &models.Category{
		APIID: "",
	})
	s.Equal(repo.ErrModelNotFound, err)
}

func (s *CategoriesSuite) TestDelete() {
	ctgID := utils.CreateAPIID()
	s.repo.Create(s.user, &models.Category{
		APIID: ctgID,
		Name:  "news",
	})

	err := s.repo.Delete(s.user, ctgID)
	s.NoError(err)

	_, found := s.repo.CategoryWithID(s.user, ctgID)
	s.False(found)
}

func (s *CategoriesSuite) TestDeleteMissing() {
	err := s.repo.Delete(s.user, "bogus")
	s.Equal(repo.ErrModelNotFound, err)
}

func (s *CategoriesSuite) TestMark() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test_ctg",
	}
	s.repo.Create(s.user, &ctg)

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

	err := s.repo.Mark(s.user, ctg.APIID, models.MarkerRead)
	s.NoError(err)

	entries, _ := NewEntries(s.db).ListFromCategory(s.user, ctg.APIID, "", 5, false, models.MarkerRead)
	s.Len(entries, 5)
}

func (s *CategoriesSuite) TestMarkUnknownCategory() {
	err := s.repo.Mark(s.user, "bogus", models.MarkerRead)
	s.Equal(repo.ErrModelNotFound, err)
}

func (s *CategoriesSuite) TestAddFeed() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "tech",
	}
	s.repo.Create(s.user, &ctg)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     ctg,
	}
	s.db.db.Model(s.user).Association("Feeds").Append(&feed)

	s.NoError(s.repo.AddFeed(s.user, feed.APIID, ctg.APIID))

	feeds, _ := s.repo.Feeds(s.user, ctg.APIID, "", 1)
	s.Require().Len(feeds, 1)
	s.Equal(feed.Title, feeds[0].Title)
	s.Equal(feed.APIID, feeds[0].APIID)
}

func (s *CategoriesSuite) TestCategoryStats() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "news",
	}

	s.repo.Create(s.user, &ctg)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	s.db.db.Model(s.user).Association("Feeds").Append(&feed)
	s.db.db.Model(&ctg).Association("Feeds").Append(&feed)

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
		s.db.db.Model(&feed).Association("Entries").Append(&entry)
	}

	stats, err := s.repo.Stats(s.user, ctg.APIID)
	s.NoError(err)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(2, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *CategoriesSuite) SetupTest() {
	s.db = NewDB("sqlite3", ":memory:")

	s.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test_ctgs",
	}

	s.db.db.Create(s.user)

	s.repo = NewCategories(s.db)
}

func (s *CategoriesSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestCategoriesSuite(t *testing.T) {
	suite.Run(t, new(CategoriesSuite))
}
