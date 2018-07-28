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

package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"sort"

	"github.com/labstack/echo"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

func (t *ServerTestSuite) TestOPMLExport() {
	ctg := database.NewCategory("Test", t.user)

	feed, err := database.NewFeedWithCategory(
		"Example", "example.com", ctg.APIID, t.user,
	)
	t.Require().NoError(err)

	req := httptest.NewRequest(echo.GET, "/", nil)
	req.Header.Set("Accept", "application/xml")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/export")

	t.NoError(t.server.Export(c))
	t.Equal(http.StatusOK, t.rec.Code)

	var exp models.OPML
	t.NoError(xml.Unmarshal(t.rec.Body.Bytes(), &exp))

	t.NotEqual(sort.Search(len(exp.Body.Items), func(i int) bool {
		item := exp.Body.Items[i]
		return item.Title == ctg.Name && item.Items[0].Title == feed.Title
	}), len(exp.Body.Items))
}
