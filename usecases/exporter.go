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
	"github.com/jmartinezhern/syndication/database"
	"github.com/jmartinezhern/syndication/models"
)

type (
	// ExporterUsecase is an interface that wraps the basic
	// export functions.
	ExporterUsecase interface {
		Export(user models.User) ([]byte, error)
	}

	// An OPMLExporter represents an exporter for the OPML 2.0 format
	// define by http://dev.opml.org/spec2.html.
	OPMLExporter struct{}
)

// Export categories and feeds to data in OPML 2.0 format.
func (i OPMLExporter) Export(user models.User) ([]byte, error) {
	ctgs := database.Categories(user)

	for idx, ctg := range ctgs {
		ctg.Feeds = database.CategoryFeeds(ctg.APIID, user)
		ctgs[idx] = ctg
	}

	b := models.OPML{
		Body: models.OPMLBody{},
	}

	for _, ctg := range ctgs {
		items := make([]models.OPMLOutline, len(ctg.Feeds))

		for idx, feed := range ctg.Feeds {
			items[idx] = models.OPMLOutline{
				Title:   feed.Title,
				Text:    feed.Title,
				Type:    "rss",
				XMLUrl:  feed.Subscription,
				HTMLUrl: feed.Subscription,
			}

		}

		if ctg.Name != models.Uncategorized {
			ctgOutline := models.OPMLOutline{
				Text:  ctg.Name,
				Title: ctg.Name,
				Items: items,
			}

			b.Body.Items = append(b.Body.Items, ctgOutline)
		} else {
			b.Body.Items = append(b.Body.Items, items...)
		}
	}

	return xml.Marshal(b)
}
