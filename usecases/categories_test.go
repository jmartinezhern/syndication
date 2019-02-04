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
	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

func (t *UsecasesTestSuite) TestNewCategory() {
	_, err := t.ctgs.New("test", t.user)
	t.NoError(err)

	_, found := database.CategoryWithName("test", t.user)
	t.True(found)
}

func (t *UsecasesTestSuite) TestNewConflictingCategory() {
	database.NewCategory("test", t.user)

	_, err := t.ctgs.New("test", t.user)
	t.Equal(ErrCategoryConflicts, err)
}

func (t *UsecasesTestSuite) TestCategory() {
	ctg := database.NewCategory("test", t.user)

	_, found := t.ctgs.Category(ctg.APIID, t.user)
	t.True(found)
}

func (t *UsecasesTestSuite) TestCategories() {
	ctg := database.NewCategory("test", t.user)

	ctgs := t.ctgs.Categories(t.user)
	t.Require().Len(ctgs, 2)
	t.Equal(ctgs[1].Name, ctg.Name)
	t.Equal(ctgs[1].APIID, ctg.APIID)
}

func (t *UsecasesTestSuite) TestCategoryFeeds() {
	ctg := database.NewCategory("test", t.user)
	feed, err := database.NewFeedWithCategory("example", "example.com", ctg.APIID, t.user)
	t.Require().NoError(err)

	feeds, err := t.ctgs.Feeds(ctg.APIID, t.user)
	t.NoError(err)
	t.Len(feeds, 1)
	t.Equal(feeds[0].Title, feed.Title)
}

func (t *UsecasesTestSuite) TestMissingCategoryFeeds() {
	_, err := t.ctgs.Feeds("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *UsecasesTestSuite) TestEditCategory() {
	ctg := database.NewCategory("test", t.user)

	newCtg, err := t.ctgs.Edit("newName", ctg.APIID, t.user)
	t.NoError(err)
	t.Equal("newName", newCtg.Name)
}

func (t *UsecasesTestSuite) TestEditMissingCategory() {
	_, err := t.ctgs.Edit("bogus", "bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *UsecasesTestSuite) TestAddFeedsToCategory() {
	ctg := database.NewCategory("test", t.user)
	feed := database.NewFeed("example", "example.com", t.user)

	feeds := database.CategoryFeeds(ctg.APIID, t.user)
	t.Empty(feeds)

	t.ctgs.AddFeeds(ctg.APIID, []string{feed.APIID}, t.user)

	feeds = database.CategoryFeeds(ctg.APIID, t.user)
	t.Require().Len(feeds, 1)
	t.Equal(feeds[0].Title, feed.Title)
}

func (t *UsecasesTestSuite) TestDeleteCategory() {
	ctg := database.NewCategory("test", t.user)

	err := t.ctgs.Delete(ctg.APIID, t.user)
	t.NoError(err)

	_, found := database.CategoryWithAPIID(ctg.APIID, t.user)
	t.False(found)
}

func (t *UsecasesTestSuite) TestDeleteMissingCategory() {
	err := t.ctgs.Delete("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *UsecasesTestSuite) TestMarkCategory() {
	ctg := database.NewCategory("test", t.user)

	feed, err := database.NewFeedWithCategory(
		"Example", "example.com", ctg.APIID, t.user,
	)
	t.Require().NoError(err)

	database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)

	t.Require().Empty(database.Entries(true, models.MarkerRead, t.user))

	err = t.ctgs.Mark(ctg.APIID, models.MarkerRead, t.user)
	t.NoError(err)
	t.Len(database.Entries(true, models.MarkerRead, t.user), 1)
}

func (t *UsecasesTestSuite) TestMarkMissingCategory() {
	err := t.ctgs.Mark("bogus", models.MarkerRead, t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *UsecasesTestSuite) TestCategoryEntries() {
	ctg := database.NewCategory("test", t.user)

	feed, err := database.NewFeedWithCategory(
		"Example", "example.com", ctg.APIID, t.user,
	)
	t.Require().NoError(err)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test Entry",
		Mark:  models.MarkerUnread,
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	entries, err := t.ctgs.Entries(ctg.APIID, true, models.MarkerUnread, t.user)
	t.NoError(err)
	t.Len(entries, 1)
	t.Equal(entries[0].Title, entry.Title)
}

func (t *UsecasesTestSuite) TestEntriesForMissingCategory() {
	_, err := t.ctgs.Entries("bogus", true, models.MarkerAny, t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}

func (t *UsecasesTestSuite) TestCategoryStats() {
	ctg := database.NewCategory("test", t.user)
	_, err := t.ctgs.Stats(ctg.APIID, t.user)
	t.NoError(err)
}

func (t *UsecasesTestSuite) TestMissingCategoryStats() {
	_, err := t.ctgs.Stats("bogus", t.user)
	t.EqualError(err, ErrCategoryNotFound.Error())
}
