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
	_, err := t.service.New("test", t.user)
	t.NoError(err)

	_, found := t.service.ctgsRepo.CategoryWithName(t.user, "test")
	t.True(found)
}

func (t *CategoriesSuite) TestNewConflictingCategory() {
	t.service.ctgsRepo.Create(t.user, &models.Category{
		Name: "test",
	})

	_, err := t.service.New("test", t.user)
	t.EqualError(err, ErrCategoryConflicts.Error())
}

func (t *CategoriesSuite) TestCategory() {
	ctgID := utils.CreateAPIID()
	t.service.ctgsRepo.Create(t.user, &models.Category{
		APIID: ctgID,
		Name:  "test",
	})

	_, found := t.service.Category(ctgID, t.user)
	t.True(found)
}

func (t *CategoriesSuite) TestCategories() {
	ctgID := utils.CreateAPIID()
	t.service.ctgsRepo.Create(t.user, &models.Category{
		APIID: ctgID,
		Name:  "test",
	})

	ctgs, _ := t.service.Categories("", 2, t.user)
	t.Require().Len(ctgs, 1)
	t.Equal("test", ctgs[0].Name)
	t.Equal(ctgID, ctgs[0].APIID)
}

func (t *CategoriesSuite) TestCategoryFeeds() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}

	sql.NewFeeds(t.db).Create(t.user, &feed)

	feeds, _ := t.service.Feeds(ctg.APIID, "", 1, t.user)
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestUncategorized() {
	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
	}

	sql.NewFeeds(t.db).Create(t.user, &feed)

	feeds, _ := t.service.Uncategorized("", 1, t.user)
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestEditCategory() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	ctg.Name = "newName"

	newCtg, err := t.service.Update("newName", ctg.APIID, t.user)
	t.NoError(err)
	t.Equal("newName", newCtg.Name)
}

func (t *CategoriesSuite) TestEditMissingCategory() {
	_, err := t.service.Update("bogus", "bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestAddFeedsToCategory() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}
	sql.NewFeeds(t.db).Create(t.user, &feed)

	t.service.AddFeeds(ctg.APIID, []string{feed.APIID}, t.user)

	feeds, _ := t.service.ctgsRepo.Feeds(t.user, ctg.APIID, "", 1)
	t.Require().Len(feeds, 1)
	t.Equal(feed.Title, feeds[0].Title)
}

func (t *CategoriesSuite) TestDeleteCategory() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	err := t.service.Delete(ctg.APIID, t.user)
	t.NoError(err)

	_, found := t.service.ctgsRepo.CategoryWithID(t.user, ctg.APIID)
	t.False(found)
}

func (t *CategoriesSuite) TestDeleteMissingCategory() {
	err := t.service.Delete("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestMarkCategory() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	feed := models.Feed{
		APIID:    utils.CreateAPIID(),
		Title:    "example",
		Category: ctg,
	}
	sql.NewFeeds(t.db).Create(t.user, &feed)

	entriesRepo := sql.NewEntries(t.db)
	entriesRepo.Create(t.user, &models.Entry{
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	})

	err := t.service.Mark(ctg.APIID, models.MarkerRead, t.user)
	t.NoError(err)

	entries, _ := entriesRepo.List(t.user, "", 2, true, models.MarkerRead)
	t.Len(entries, 1)
}

func (t *CategoriesSuite) TestMarkMissingCategory() {
	err := t.service.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryEntries() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
		Category:     ctg,
	}
	sql.NewFeeds(t.db).Create(t.user, &feed)

	entriesRepo := sql.NewEntries(t.db)
	entriesRepo.Create(t.user, &models.Entry{
		APIID: utils.CreateAPIID(),
		Title: "Test Entries",
		Mark:  models.MarkerUnread,
		Feed:  feed,
	})

	entries, _, err := t.service.Entries(ctg.APIID, "", 1, true, models.MarkerUnread, t.user)
	t.Require().Len(entries, 1)
	t.NoError(err)
	t.Equal("Test Entries", entries[0].Title)
}

func (t *CategoriesSuite) TestEntriesForMissingCategory() {
	_, _, err := t.service.Entries("bogus", "", 1, true, models.MarkerAny, t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryStats() {
	ctg := models.Category{
		APIID: utils.CreateAPIID(),
		Name:  "test",
	}
	t.service.ctgsRepo.Create(t.user, &ctg)

	_, err := t.service.Stats(ctg.APIID, t.user)
	t.NoError(err)
}

func (t *CategoriesSuite) TestMissingCategoryStats() {
	_, err := t.service.Stats("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")
	t.service = NewCategoriesService(sql.NewCategories(t.db), sql.NewEntries(t.db))

	t.user = &models.User{
		APIID:    utils.CreateAPIID(),
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
