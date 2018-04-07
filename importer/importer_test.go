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

package importer

import (
	"io/ioutil"
	"testing"

	"github.com/varddum/syndication/models"

	"github.com/stretchr/testify/suite"
)

type (
	ImporterTestSuite struct {
		suite.Suite
	}
)

func (s *ImporterTestSuite) TestOPMLImport() {
	importer := NewOPMLImporter()

	data, err := ioutil.ReadFile("sample.opml")
	s.Require().Nil(err)

	items := importer.Import(data)
	s.Require().Len(items, 16)
	s.Equal("Sports", items[0].Category.Name)
}

func (s *ImporterTestSuite) TestOPMLExport() {
	encoder := NewOPMLImporter()

	ctgs := []models.Category{
		models.Category{
			Name: "Sports",
		},
		models.Category{
			Name: models.Uncategorized,
		},
	}

	ctgs[0].Feeds = []models.Feed{
		models.Feed{
			Title:        "Baseball",
			Subscription: "http://example.com/baseball",
		},
	}

	ctgs[1].Feeds = []models.Feed{
		models.Feed{
			Title:        "Baskeball",
			Subscription: "http://example.com/basketball",
		},
	}

	data, err := encoder.Export(ctgs)
	s.Require().Nil(err)

	feeds := encoder.Import(data)
	s.Require().Len(feeds, 2)

	s.Equal("Sports", feeds[0].Category.Name)
	s.Empty(feeds[1].Category.Name)
}

func TestImporterTestSuite(t *testing.T) {
	suite.Run(t, new(ImporterTestSuite))
}
