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
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo/sql"
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

	importer OPMLImporter
	db       *sql.DB
	user     *models.User
}

func (t *ImporterSuite) TestOPMLImporter() {
	err := t.importer.Import([]byte(opml), t.user)
	t.NoError(err)

	ctg, found := t.importer.ctgsRepo.CategoryWithName(t.user, "Test")
	t.Require().True(found)

	ctgFeeds, _ := t.importer.ctgsRepo.Feeds(t.user, ctg.APIID, "", 10)
	t.Require().Len(ctgFeeds, 1)
	t.Equal(ctgFeeds[0].Title, "Example")

	feeds, _ := t.importer.feedsRepo.List(t.user, "", 2)

	t.NotZero(sort.Search(len(feeds), func(i int) bool {
		return feeds[i].Title == "Empty"
	}))
}

func (t *ImporterSuite) SetupTest() {
	t.db = sql.NewDB("sqlite3", ":memory:")

	t.user = &models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}

	sql.NewUsers(t.db).Create(t.user)

	t.importer = NewOPMLImporter(sql.NewCategories(t.db), sql.NewFeeds(t.db))
}

func (t *ImporterSuite) TearDownTest() {
	err := t.db.Close()
	t.NoError(err)
}

func TestImporter(t *testing.T) {
	suite.Run(t, new(ImporterSuite))
}
