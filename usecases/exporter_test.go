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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type ExporterSuite struct {
	suite.Suite

	user     models.User
	unctgCtg models.Category
}

func (t *ExporterSuite) TestOPMLExporter() {
	exporter := OPMLExporter{}

	ctgID := utils.CreateAPIID()
	database.CreateCategory(&models.Category{
		APIID: ctgID,
		Name:  "test",
	}, t.user)

	err := database.CreateFeed(&models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Empty",
		Subscription: "example.com",
	}, t.unctgCtg.APIID, t.user)
	t.NoError(err)

	feed := models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Empty",
		Subscription: "example.com",
	}
	err = database.CreateFeed(&feed, ctgID, t.user)
	t.NoError(err)

	data, err := exporter.Export(t.user)
	t.NoError(err)

	b := models.OPML{}
	t.NoError(xml.Unmarshal(data, &b))

	t.Require().Len(b.Body.Items, 2)
	t.Require().Len(b.Body.Items[1].Items, 1)
	t.NotZero(sort.Search(len(b.Body.Items), func(i int) bool {
		return b.Body.Items[i].Title == "Empty"
	}))
	t.NotZero(sort.Search(len(b.Body.Items), func(i int) bool {
		return b.Body.Items[i].Title == "test" && b.Body.Items[i].Items[0].Title == feed.Title
	}))
}

func (t *ExporterSuite) SetupTest() {
	err := database.Init("sqlite3", ":memory:")
	t.Require().NoError(err)

	t.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "gopher",
	}
	database.CreateUser(&t.user)

	t.unctgCtg = models.Category{
		APIID: utils.CreateAPIID(),
		Name:  models.Uncategorized,
	}
	database.CreateCategory(&t.unctgCtg, t.user)
}

func (t *ExporterSuite) TearDownTest() {
	err := database.Close()
	t.NoError(err)
}

func TestExporter(t *testing.T) {
	suite.Run(t, new(ExporterSuite))
}
