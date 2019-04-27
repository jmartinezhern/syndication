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
	"encoding/xml"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/utils"
)

type ExporterSuite struct {
	suite.Suite

	service OPMLExporter
	user    *models.User
	repo    repo.Categories
	db      *sql.DB
}

func (t *ExporterSuite) TestOPMLExporter() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}
	t.repo.Create(t.user.ID, &ctg)

	feedsRepo := sql.NewFeeds(t.db)
	unCtgFeed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Uncategorized",
		Subscription: "example.com",
	}
	feedsRepo.Create(t.user.ID, &unCtgFeed)

	ctgFeed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Categorized",
		Subscription: "example.com",
		Category:     ctg,
	}
	feedsRepo.Create(t.user.ID, &ctgFeed)

	data, err := t.service.Export(t.user.ID)
	t.NoError(err)

	b := models.OPML{}
	t.NoError(xml.Unmarshal(data, &b))

	t.Require().Len(b.Body.Items, 2)
	t.NotZero(sort.Search(len(b.Body.Items), func(i int) bool {
		return b.Body.Items[i].Title == unCtgFeed.Title
	}))
	t.NotZero(sort.Search(len(b.Body.Items), func(i int) bool {
		return b.Body.Items[i].Title == ctg.Name && b.Body.Items[i].Items[0].Title == ctgFeed.Title
	}))
}

func (t *ExporterSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")
	t.repo = sql.NewCategories(t.db)

	t.service = NewOPMLExporter(t.repo)

	t.user = &models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}
	sql.NewUsers(t.db).Create(t.user)
}

func (t *ExporterSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestExporter(t *testing.T) {
	suite.Run(t, new(ExporterSuite))
}
