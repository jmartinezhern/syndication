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
	"encoding/xml"
	"sort"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

func (t *UsecasesTestSuite) TestOPMLExporter() {
	exporter := OPMLExporter{}

	ctg := database.NewCategory("Test", t.user)
	database.NewFeed("Empty", "empty.com", t.user)
	feed, err := database.NewFeedWithCategory("Example", "example.com", ctg.APIID, t.user)
	t.NoError(err)

	data, err := exporter.Export(t.user)
	t.NoError(err)

	b := models.OPML{}
	t.NoError(xml.Unmarshal(data, &b))

	t.Len(b.Body.Items, 2)
	t.Len(b.Body.Items[1].Items, 1)
	t.NotZero(sort.Search(len(b.Body.Items), func(i int) bool {
		return b.Body.Items[i].Title == "Empty"
	}))
	t.NotZero(sort.Search(len(b.Body.Items), func(i int) bool {
		return b.Body.Items[i].Title == ctg.Name && b.Body.Items[i].Items[0].Title == feed.Title
	}))
}
