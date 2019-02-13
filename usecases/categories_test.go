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

package usecases

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type CategoriesSuite struct {
	suite.Suite

	ctgs Category
	user models.User
}

func (t *CategoriesSuite) TestNewCategory() {
	_, err := t.ctgs.New("test", t.user)
	t.NoError(err)

	_, found := database.CategoryWithName("test", t.user)
	t.True(found)
}

func (t *CategoriesSuite) TestNewConflictingCategory() {
	database.CreateCategory(&models.Category{
		Name: "test",
	}, t.user)

	_, err := t.ctgs.New("test", t.user)
	t.Equal(ErrCategoryConflicts, err)
}

func (t *CategoriesSuite) TestCategory() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	_, found := t.ctgs.Category(ctgID, t.user)
	t.True(found)
}

func (t *CategoriesSuite) TestCategories() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	ctgs, _ := t.ctgs.Categories(t.user, "", 2)
	t.Require().Len(ctgs, 1)
	t.Equal("test", ctgs[0].Name)
	t.Equal(ctgID, ctgs[0].APIID)
}

func (t *CategoriesSuite) TestCategoryFeeds() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
	}
	err := database.CreateFeed(&feed, ctgID, t.user)
	t.Require().NoError(err)

	feeds, err := t.ctgs.Feeds(ctgID, t.user)
	t.NoError(err)
	t.Len(feeds, 1)
	t.Equal(feeds[0].Title, feed.Title)
}

func (t *CategoriesSuite) TestMissingCategoryFeeds() {
	_, err := t.ctgs.Feeds("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestEditCategory() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	newCtg, err := t.ctgs.Edit("newName", ctgID, t.user)
	t.NoError(err)
	t.Equal("newName", newCtg.Name)
}

func (t *CategoriesSuite) TestEditMissingCategory() {
	_, err := t.ctgs.Edit("bogus", "bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestAddFeedsToCategory() {
	unctgCtgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: unctgCtgID,
		Name:  models.Uncategorized,
	}, t.user)
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
	}
	err := database.CreateFeed(&feed, unctgCtgID, t.user)
	t.Require().NoError(err)

	feeds := database.CategoryFeeds(ctgID, t.user)
	t.Empty(feeds)

	t.ctgs.AddFeeds(ctgID, []string{feed.APIID}, t.user)

	feeds = database.CategoryFeeds(ctgID, t.user)
	t.Require().Len(feeds, 1)
	t.Equal(feeds[0].Title, feed.Title)
}

func (t *CategoriesSuite) TestDeleteCategory() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	err := t.ctgs.Delete(ctgID, t.user)
	t.NoError(err)

	_, found := database.CategoryWithAPIID(ctgID, t.user)
	t.False(found)
}

func (t *CategoriesSuite) TestDeleteMissingCategory() {
	err := t.ctgs.Delete("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestMarkCategory() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
	}
	err := database.CreateFeed(&feed, ctgID, t.user)
	t.Require().NoError(err)

	database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)

	entries, _ := database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Require().Empty(entries)

	err = t.ctgs.Mark(ctgID, models.MarkerRead, t.user)
	t.NoError(err)

	entries, _ = database.Entries(true, models.MarkerRead, "", 2, t.user)
	t.Len(entries, 1)
}

func (t *CategoriesSuite) TestMarkMissingCategory() {
	err := t.ctgs.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryEntries() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "example",
		Subscription: "example.com",
	}
	err := database.CreateFeed(&feed, ctgID, t.user)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	entries, err := t.ctgs.Entries(ctgID, true, models.MarkerUnread, t.user)
	t.NoError(err)
	t.Len(entries, 1)
	t.Equal(entries[0].Title, entry.Title)
}

func (t *CategoriesSuite) TestEntriesForMissingCategory() {
	_, err := t.ctgs.Entries("bogus", true, models.MarkerAny, t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) TestCategoryStats() {
	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	_, err := t.ctgs.Stats(ctgID, t.user)
	t.NoError(err)
}

func (t *CategoriesSuite) TestMissingCategoryStats() {
	_, err := t.ctgs.Stats("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *CategoriesSuite) SetupTest() {
	t.ctgs = new(CategoryUsecase)

	err := database.Init("sqlite3", ":memory:")
	t.Require().NoError(err)

	t.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}
	database.CreateUser(&t.user)
}

func (t *CategoriesSuite) TearDownTest() {
	err := database.Close()
	t.NoError(err)
}

func TestCategories(t *testing.T) {
	suite.Run(t, new(CategoriesSuite))
}
