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

package rest

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	ExporterControllerSuite struct {
		suite.Suite

		e          *echo.Echo
		db         *sql.DB
		user       *models.User
		controller *ExporterController
	}
)

func (c *ExporterControllerSuite) TestOPMLExport() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "Test",
	}
	sql.NewCategories(c.db).Create(c.user.ID, &ctg)

	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "example.com",
		Subscription: "https://example.com",
		Category:     ctg,
	}
	sql.NewFeeds(c.db).Create(c.user.ID, &feed)

	req := httptest.NewRequest(echo.GET, "/", nil)
	req.Header.Set("Accept", "application/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/export")

	c.NoError(c.controller.Export(ctx))
	c.Equal(http.StatusOK, rec.Code)

	var exp models.OPML

	c.NoError(xml.Unmarshal(rec.Body.Bytes(), &exp))
	c.NotEqual(sort.Search(len(exp.Body.Items), func(i int) bool {
		item := exp.Body.Items[i]
		return item.Title == ctg.Name && item.Items[0].Title == feed.Title
	}), len(exp.Body.Items))
}

func (c *ExporterControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.db = sql.NewDB("sqlite3", ":memory:")
	ctgsRepo := sql.NewCategories(c.db)
	sql.NewUsers(c.db).Create(c.user)

	exporters := Exporters{
		"application/xml": services.NewOPMLExporter(ctgsRepo),
	}
	c.controller = NewExporterController(exporters, c.e)
}

func (c *ExporterControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}
func TestExporterControllerSuite(t *testing.T) {
	suite.Run(t, new(ExporterControllerSuite))
}
