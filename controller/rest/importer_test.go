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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	ImporterControllerSuite struct {
		suite.Suite

		e          *echo.Echo
		db         *sql.DB
		user       *models.User
		controller *ImporterController
	}
)

const (
	data = `
<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0">
		<body>
			<outline text="Sports" title="Sports">
				<outline
					type="rss"
					text="Basketball"
					title="Basketball"
					xmlUrl="http://example.com/basketball"
					htmlUrl="http://example.com/basketball"
					/>
			</outline>
			<outline
				type="rss"
				text="Baseball"
				title="Baseball"
				xmlUrl="http://example.com/baseball"
				htmlUrl="http://example.com/baseball"
				/>
		</body>
	</opml>
	`
)

func (c *ImporterControllerSuite) TestOPMLImport() {
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(data))
	req.Header.Set("Content-Type", "application/xml")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/import")

	c.NoError(c.controller.Import(ctx))
	c.Equal(http.StatusNoContent, rec.Code)

	ctgsRepo := sql.NewCategories(c.db)
	ctgs, _ := ctgsRepo.List(c.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
	})
	c.Require().Len(ctgs, 1)
	c.Equal("Sports", ctgs[0].Name)
	c.NotEmpty(ctgs[0].ID)

	feeds, _ := ctgsRepo.Feeds(c.user.ID, models.Page{
		FilterID:       ctgs[0].ID,
		ContinuationID: "",
		Count:          1,
	})
	c.Require().Len(feeds, 1)
	c.Equal("Basketball", feeds[0].Title)
	c.Equal("http://example.com/basketball", feeds[0].Subscription)

	unctgsFeeds, _ := ctgsRepo.Uncategorized(c.user.ID, models.Page{
		ContinuationID: "",
		Count:          1,
	})
	c.Require().Len(unctgsFeeds, 1)
	c.Equal("Baseball", unctgsFeeds[0].Title)
	c.Equal("http://example.com/baseball", unctgsFeeds[0].Subscription)
}

func (c *ImporterControllerSuite) SetupTest() {
	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.db = sql.NewDB("sqlite3", ":memory:")
	ctgsRepo := sql.NewCategories(c.db)
	sql.NewUsers(c.db).Create(c.user)

	importers := Importers{
		"application/xml": services.NewOPMLImporter(ctgsRepo, sql.NewFeeds(c.db)),
	}
	c.controller = NewImporterController(importers, c.e)
}

func (c *ImporterControllerSuite) TearDownTest() {
	err := c.db.Close()
	c.NoError(err)
}
func TestImporterControllerSuite(t *testing.T) {
	suite.Run(t, new(ImporterControllerSuite))
}
