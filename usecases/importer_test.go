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
	"sort"

	"github.com/jmartinezhern/syndication/database"
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

func (t *UsecasesTestSuite) TestOPMLImporter() {
	importer := OPMLImporter{}

	importer.Import([]byte(opml), t.user)

	ctg, found := database.CategoryWithName("Test", t.user)
	t.True(found)

	ctgFeeds := database.CategoryFeeds(ctg.APIID, t.user)
	t.Len(ctgFeeds, 1)
	t.Equal(ctgFeeds[0].Title, "Example")

	feeds := database.Feeds(t.user)

	t.NotZero(sort.Search(len(feeds), func(i int) bool {
		return feeds[i].Title == "Empty"
	}))
}
