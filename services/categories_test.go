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

package services

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type CategoriesSuite struct {
	suite.Suite

	service CategoriesService

	db   *sql.DB
	user *models.User
}

func (t *CategoriesSuite) TestNewCategory() {
	_, err := t.service.New("test", t.user.ID)
	t.NoError(err)

	_, found := t.service.ctgsRepo.CategoryWithName(t.user.ID, "test")
	t.True(found)
}

func (t *CategoriesSuite) TestNewConflictingCategory() {
	t.service.ctgsRepo.Create(t.user.ID, &models.Category{
		Name: "test",
	})

	_, err := t.service.New("test", t.user.ID)
	t.EqualError(err, ErrCategoryConflicts.Error())
}

func (t *CategoriesSuite) TestCategory() {
	ctgID := utils.CreateID()
	t.service.ctgsRepo.Create(t.user.ID, &models.Category{
		ID:   ctgID,
		Name: "test",
	})

	_, found := t.service.Category(ctgID, t.user.ID)
	t.True(found)
}

func (t *CategoriesSuite) TestCategories() {
	ctgID := utils.CreateID()
	t.service.ctgsRepo.Create(t.user.ID, &models.Category{
		ID:   ctgID,
		Name: "test",
	})

	ctgs, _ := t.service.Categories("", 2, t.user.ID)
	t.Require().Len(ctgs, 1)
	t.Equal("test", ctgs[0].Name)
	t.Equal(ctgID, ctgs[0].ID)
}

func (t *CategoriesSuite) TestCategoryFeeds() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}

	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	feeds, _ := t.service.Feeds(ctg.ID, "", 1, t.user.ID)
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

	feeds, _ := t.service.Uncategorized("", 1, t.user.ID)
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestEditCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

	ctg.Name = "newName"

	newCtg, err := t.service.Update("newName", ctg.ID, t.user.ID)
	t.NoError(err)
	t.Equal("newName", newCtg.Name)
}

func (t *CategoriesSuite) TestEditMissingCategory() {
	_, err := t.service.Update("bogus", "bogus", t.user.ID)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestAddFeedsToCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}
	sql.NewFeeds(t.db).Create(t.user.ID, &feed)

	t.service.AddFeeds(ctg.ID, []string{feed.ID}, t.user.ID)

	feeds, _ := t.service.ctgsRepo.Feeds(t.user.ID, ctg.ID, "", 1)
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestDeleteCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

	err := t.service.Delete(ctg.ID, t.user.ID)
	t.NoError(err)

	_, found := t.service.ctgsRepo.CategoryWithID(t.user.ID, ctg.ID)
	t.False(found)
}

func (t *CategoriesSuite) TestDeleteMissingCategory() {
	err := t.service.Delete("bogus", t.user.ID)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestMarkCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

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

	err := t.service.Mark(ctg.ID, models.MarkerRead, t.user.ID)
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
	err := t.service.Mark("bogus", models.MarkerRead, t.user.ID)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryEntries() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

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

	entries, _, err := t.service.Entries(ctg.ID, t.user.ID, models.Page{
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
	_, _, err := t.service.Entries("bogus", t.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
		Newest:         true,
		Marker:         models.MarkerAny,
	})
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryStats() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.service.ctgsRepo.Create(t.user.ID, &ctg)

	_, err := t.service.Stats(ctg.ID, t.user.ID)
	t.NoError(err)
}

func (t *CategoriesSuite) TestMissingCategoryStats() {
	_, err := t.service.Stats("bogus", t.user.ID)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")
	t.service = NewCategoriesService(sql.NewCategories(t.db), sql.NewEntries(t.db))

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

func TestCategories(t *testing.T) {
	suite.Run(t, new(CategoriesSuite))
}
