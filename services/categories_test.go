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

package services_test

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type CategoriesSuite struct {
	suite.Suite

	service services.CategoriesService

	db       *gorm.DB
	ctgsRepo repo.Categories
	user     *models.User
}

func (t *CategoriesSuite) TestNewCategory() {
	_, err := t.service.New(t.user.ID, "test")
	t.NoError(err)

	_, found := t.ctgsRepo.CategoryWithName(t.user.ID, "test")
	t.True(found)
}

func (t *CategoriesSuite) TestNewConflictingCategory() {
	t.ctgsRepo.Create(t.user.ID, &models.Category{
		Name: "test",
	})

	_, err := t.service.New(t.user.ID, "test")
	t.EqualError(err, services.ErrCategoryConflicts.Error())
}

func (t *CategoriesSuite) TestCategory() {
	ctgID := utils.CreateID()

	t.ctgsRepo.Create(t.user.ID, &models.Category{
		ID:   ctgID,
		Name: "test",
	})

	_, found := t.service.Category(t.user.ID, ctgID)
	t.True(found)
}

func (t *CategoriesSuite) TestCategories() {
	ctgID := utils.CreateID()

	t.ctgsRepo.Create(t.user.ID, &models.Category{
		ID:   ctgID,
		Name: "test",
	})

	ctgs, _ := t.service.Categories(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
	})
	t.Require().Len(ctgs, 1)
	t.Equal("test", ctgs[0].Name)
	t.Equal(ctgID, ctgs[0].ID)
}

func (t *CategoriesSuite) TestCategoryFeeds() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}

	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	feeds, _ := t.service.Feeds(t.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          1,
	})
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestUncategorized() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "example.com",
	}

	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	feeds, _ := t.service.Uncategorized(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
	})
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestEditCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	ctg.Name = "newName"

	newCtg, err := t.service.Update(t.user.ID, ctg.ID, "newName")
	t.NoError(err)
	t.Equal("newName", newCtg.Name)
}

func (t *CategoriesSuite) TestEditMissingCategory() {
	_, err := t.service.Update(t.user.ID, "bogus", "bogus")
	t.EqualError(err, services.ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestAddFeedsToCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}
	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	t.service.AddFeeds(t.user.ID, ctg.ID, []string{feed.ID})

	feeds, _ := t.ctgsRepo.Feeds(t.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          1,
	})
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestDeleteCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	err := t.service.Delete(t.user.ID, ctg.ID)
	t.NoError(err)

	_, found := t.ctgsRepo.CategoryWithID(t.user.ID, ctg.ID)
	t.False(found)
}

func (t *CategoriesSuite) TestDeleteMissingCategory() {
	err := t.service.Delete(t.user.ID, "bogus")
	t.EqualError(err, services.ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestMarkCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	feed := models.Feed{
		ID:       utils.CreateID(),
		Title:    "example",
		Category: ctg,
	}
	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	entriesRepo := sql.NewEntries(t.db)
	entriesRepo.Create(t.user.ID, &models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	})

	err := t.service.Mark(t.user.ID, ctg.ID, models.MarkerRead)
	t.NoError(err)

	entries, _ := entriesRepo.List(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
		Newest:         true,
		Marker:         models.MarkerRead,
	})
	t.Len(entries, 1)
}

func (t *CategoriesSuite) TestMarkMissingCategory() {
	err := t.service.Mark(t.user.ID, "bogus", models.MarkerRead)
	t.EqualError(err, services.ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryEntries() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}
	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	entriesRepo := sql.NewEntries(t.db)
	entriesRepo.Create(t.user.ID, &models.Entry{
		ID:    utils.CreateID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	})

	entries, _, err := t.service.Entries(t.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          1,
		Newest:         true,
		Marker:         models.MarkerUnread,
	})
	t.Require().Len(entries, 1)
	t.NoError(err)
	t.Equal("Test Entries", entries[0].Title)
}

func (t *CategoriesSuite) TestEntriesForMissingCategory() {
	_, _, err := t.service.Entries(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
		Newest:         true,
		Marker:         models.MarkerAny,
	})
	t.EqualError(err, services.ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryStats() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.ctgsRepo.Create(t.user.ID, &ctg)

	_, err := t.service.Stats(t.user.ID, ctg.ID)
	t.NoError(err)
}

func (t *CategoriesSuite) TestMissingCategoryStats() {
	_, err := t.service.Stats(t.user.ID, "bogus")
	t.EqualError(err, services.ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) SetupTest() {
	var err error

	t.db, err = gorm.Open("sqlite3", ":memory:")
	t.Require().NoError(err)

	sql.AutoMigrateTables(t.db)

	t.ctgsRepo = sql.NewCategories(t.db)

	t.service = services.NewCategoriesService(t.ctgsRepo, sql.NewEntries(t.db))

	t.user = &models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}
	sql.NewUsers(t.db).Create(t.user)
}

func (t *CategoriesSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestCategoriesSuite(t *testing.T) {
	suite.Run(t, new(CategoriesSuite))
}
