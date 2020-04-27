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

package services_test

import (
	"sort"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

const opml = `
<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
    <head>
        <title>Test Importer</title>
    </head>
    <body>
        <outline text="Test" title="Test">
			<outline type="rss"
					 text="Example"
					 title="Example"
					 xmlUrl="example.com" 
					 htmlUrl="example.com"/>
        </outline>
		<outline type="rss"
				 text="Empty"
				 title="Empty"
				 xmlUrl="empty.com"
				 htmlUrl="empty.com"/>
    </body>
</opml>
`

type ImporterSuite struct {
	suite.Suite

	importer  services.OPMLImporter
	ctgsRepo  repo.Categories
	feedsRepo repo.Feeds
	db        *gorm.DB
	user      *models.User
}

func (t *ImporterSuite) TestOPMLImporter() {
	err := t.importer.Import([]byte(opml), t.user.ID)
	t.NoError(err)

	ctg, found := t.ctgsRepo.CategoryWithName(t.user.ID, "Test")
	t.Require().True(found)

	ctgFeeds, _ := t.ctgsRepo.Feeds(t.user.ID, models.Page{
		FilterID:       ctg.ID,
		ContinuationID: "",
		Count:          10,
	})
	t.Require().Len(ctgFeeds, 1)
	t.Equal(ctgFeeds[0].Title, "Example")

	feeds, _ := t.feedsRepo.List(t.user.ID, models.Page{
		ContinuationID: "",
		Count:          2,
	})

	t.NotZero(sort.Search(len(feeds), func(i int) bool {
		return feeds[i].Title == "Empty"
	}))
}

func (t *ImporterSuite) SetupTest() {
	var err error

	t.db, err = gorm.Open("sqlite3", ":memory:")
	t.Require().NoError(err)

	sql.AutoMigrateTables(t.db)

	t.user = &models.User{
		ID:       utils.CreateID(),
		Username: "gopher",
	}

	sql.NewUsers(t.db).Create(t.user)

	t.ctgsRepo = sql.NewCategories(t.db)
	t.feedsRepo = sql.NewFeeds(t.db)

	t.importer = services.NewOPMLImporter(t.ctgsRepo, t.feedsRepo)
}

func (t *ImporterSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestImporter(t *testing.T) {
	suite.Run(t, new(ImporterSuite))
}
