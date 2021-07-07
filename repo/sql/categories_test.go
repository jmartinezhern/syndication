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

package sql_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type CategoriesSuite struct {
	suite.Suite

	repo repo.Categories
	db   *gorm.DB
	user *models.User
}

func (s *CategoriesSuite) TestCreate() {
	ctgID := utils.CreateID()

	s.repo.Create(s.user.ID, &models.Category{
		ID:   ctgID,
		Name: "news",
	})

	ctg, found := s.repo.CategoryWithID(s.user.ID, ctgID)
	s.True(found)
	s.Equal(ctgID, ctg.ID)
	s.Equal("news", ctg.Name)
}

func (s *CategoriesSuite) TestCategories() {
	ctgs := make([]models.Category, 5)
	for idx := range ctgs {
		ctg := models.Category{
			ID:   utils.CreateID(),
			Name: "Test Category " + strconv.Itoa(idx),
		}
		s.repo.Create(s.user.ID, &ctg)
		ctgs[idx] = ctg
	}

	cCtgs, next := s.repo.List(s.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
	})
	s.Require().Len(cCtgs, 2)
	s.Equal(ctgs[2].ID, next)
	s.Equal(ctgs[0].Name, cCtgs[0].Name)
	s.Equal(ctgs[1].Name, cCtgs[1].Name)

	cCtgs, next = s.repo.List(s.user.ID, models.Page{
		ContinuationID: next,
		Count:          3,
	})
	s.Len(next, 0)
	s.Require().Len(cCtgs, 3)
	s.Equal(ctgs[2].Name, cCtgs[0].Name)
	s.Equal(ctgs[3].Name, cCtgs[1].Name)
	s.Equal(ctgs[4].Name, cCtgs[2].Name)
}

func (s *CategoriesSuite) TestFeeds() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	s.repo.Create(s.user.ID, &ctg)

	for i := 0; i < 5; i++ {
		feed := models.Feed{
			ID:           utils.CreateID(),
			Title:        "Feed " + strconv.Itoa(i),
			Subscription: "https://example.com",
			Category:     ctg,
		}
		s.db.Model(s.user).Association("Feeds").Append(&feed)
	}

	feeds, next := s.repo.Feeds(s.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          2,
	})
	s.NotEmpty(next)
	s.Require().Len(feeds, 2)
	s.Equal("Feed 0", feeds[0].Title)
	s.Equal("Feed 1", feeds[1].Title)

	feeds, _ = s.repo.Feeds(s.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: next,
		Count:          3,
	})
	s.Require().Len(feeds, 3)
	s.Equal(feeds[0].ID, next)
	s.Equal("Feed 2", feeds[0].Title)
	s.Equal("Feed 3", feeds[1].Title)
	s.Equal("Feed 4", feeds[2].Title)
}

func (s *CategoriesSuite) TestUncategorized() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			ID:           utils.CreateID(),
			Title:        "Feed " + strconv.Itoa(i),
			Subscription: "https://example.com",
		}
		s.db.Model(s.user).Association("Feeds").Append(&feed)
	}

	feeds, next := s.repo.Uncategorized(s.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
	})
	s.NotEmpty(next)
	s.Require().Len(feeds, 2)
	s.Equal("Feed 0", feeds[0].Title)
	s.Equal("Feed 1", feeds[1].Title)

	feeds, _ = s.repo.Uncategorized(s.user.ID, models.Page{
		ContinuationID: next,
		Count:          3,
	})
	s.Require().Len(feeds, 3)
	s.Equal(feeds[0].ID, next)
	s.Equal("Feed 2", feeds[0].Title)
	s.Equal("Feed 3", feeds[1].Title)
	s.Equal("Feed 4", feeds[2].Title)
}

func (s *CategoriesSuite) TestCategoryWithName() {
	s.repo.Create(s.user.ID, &models.Category{
		Name: "test",
	})

	ctg, found := s.repo.CategoryWithName(s.user.ID, "test")
	s.True(found)
	s.Equal("test", ctg.Name)
}

func (s *CategoriesSuite) TestUpdate() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "news",
	}
	s.repo.Create(s.user.ID, &ctg)

	ctg.Name = "updates"

	err := s.repo.Update(s.user.ID, &ctg)
	s.NoError(err)

	updatedCtg, _ := s.repo.CategoryWithID(s.user.ID, ctg.ID)
	s.Equal(ctg.Name, updatedCtg.Name)
}

func (s *CategoriesSuite) TestUpdateMissing() {
	err := s.repo.Update(s.user.ID, &models.Category{
		ID: "",
	})
	s.Equal(repo.ErrModelNotFound, err)
}

func (s *CategoriesSuite) TestDelete() {
	ctgID := utils.CreateID()

	s.repo.Create(s.user.ID, &models.Category{
		ID:   ctgID,
		Name: "news",
	})

	err := s.repo.Delete(s.user.ID, ctgID)
	s.NoError(err)

	_, found := s.repo.CategoryWithID(s.user.ID, ctgID)
	s.False(found)
}

func (s *CategoriesSuite) TestDeleteMissing() {
	err := s.repo.Delete(s.user.ID, "bogus")
	s.Equal(repo.ErrModelNotFound, err)
}

func (s *CategoriesSuite) TestMark() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test_ctg",
	}
	s.repo.Create(s.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	s.db.Model(s.user).Association("Feeds").Append(&feed)
	s.db.Model(&ctg).Association("Feeds").Append(&feed)

	for i := 0; i < 5; i++ {
		entry := models.Entry{
			ID:        utils.CreateID(),
			Title:     "Entry " + strconv.Itoa(i),
			Author:    "John Doe",
			Link:      "http://example.com",
			Mark:      models.MarkerUnread,
			Published: time.Now(),
		}

		s.db.Model(s.user).Association("Entries").Append(&entry)
		s.db.Model(&feed).Association("Entries").Append(&entry)
	}

	err := s.repo.Mark(s.user.ID, ctg.ID, models.MarkerRead)
	s.NoError(err)

	entries, _ := sql.NewEntries(s.db).ListFromCategory(s.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          5,
		Newest:         false,
		Marker:         models.MarkerRead,
	})
	s.Len(entries, 5)
}

func (s *CategoriesSuite) TestMarkUnknownCategory() {
	err := s.repo.Mark(s.user.ID, "bogus", models.MarkerRead)
	s.Equal(repo.ErrModelNotFound, err)
}

func (s *CategoriesSuite) TestAddFeed() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "tech",
	}
	s.repo.Create(s.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     ctg,
	}
	s.db.Model(s.user).Association("Feeds").Append(&feed)

	s.NoError(s.repo.AddFeed(s.user.ID, feed.ID, ctg.ID))

	feeds, _ := s.repo.Feeds(s.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          1,
	})
	s.Require().Len(feeds, 1)
	s.Equal(feed.Title, feeds[0].Title)
	s.Equal(feed.ID, feeds[0].ID)
}

func (s *CategoriesSuite) TestCategoryStats() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "news",
	}

	s.repo.Create(s.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	s.db.Model(s.user).Association("Feeds").Append(&feed)
	s.db.Model(&ctg).Association("Feeds").Append(&feed)

	for i := 0; i < 10; i++ {
		var marker models.Marker

		if i < 3 {
			marker = models.MarkerRead
		} else {
			marker = models.MarkerUnread
		}

		entry := models.Entry{
			ID:        utils.CreateID(),
			Title:     "Item",
			Link:      "http://example.com",
			Mark:      marker,
			Saved:     i < 2,
			Published: time.Now(),
		}

		s.db.Model(s.user).Association("Entries").Append(&entry)
		s.db.Model(&feed).Association("Entries").Append(&entry)
	}

	stats, err := s.repo.Stats(s.user.ID, ctg.ID)
	s.NoError(err)
	s.Equal(7, stats.Unread)
	s.Equal(3, stats.Read)
	s.Equal(2, stats.Saved)
	s.Equal(10, stats.Total)
}

func (s *CategoriesSuite) SetupTest() {
	var err error

	s.db, err = gorm.Open("sqlite3", ":memory:")
	s.Require().NoError(err)

	sql.AutoMigrateTables(s.db)

	s.user = &models.User{
		ID:       utils.CreateID(),
		Username: "test_ctgs",
	}

	s.db.Create(s.user.ID)

	s.repo = sql.NewCategories(s.db)
}

func (s *CategoriesSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestCategoriesSuite(t *testing.T) {
	suite.Run(t, new(CategoriesSuite))
}
