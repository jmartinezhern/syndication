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
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

type (
	// Importer is an interface that wraps the basic
	// import functions.
	Importer interface {
		Import([]byte, models.User) error
	}

	// An OPMLImporter represents an importer for the OPML 2.0 format
	// define by http://dev.opml.org/spec2.html.
	OPMLImporter struct{}
)

func (i OPMLImporter) extractFeeds(items []models.OPMLOutline) []models.Feed {
	feeds := []models.Feed{}

	for _, outline := range items {
		if outline.Type == "rss" {
			feed := models.Feed{
				Title:        outline.Title,
				Subscription: outline.XMLUrl,
			}

			feeds = append(feeds, feed)
		} else if outline.Type == "" && len(outline.Items) > 0 {
			// We consider this a category
			ctg := models.Category{
				Name: outline.Title,
			}
			for _, ctgOutline := range outline.Items {
				feed := models.Feed{
					Title:        ctgOutline.Title,
					Subscription: ctgOutline.XMLUrl,
					Category:     ctg,
				}

				feeds = append(feeds, feed)
			}
		}
	}

	return feeds
}

// Import data that must be in a OPML 2.0 format.
func (i OPMLImporter) Import(data []byte, user models.User) error {
	b := models.OPML{}

	err := xml.Unmarshal(data, &b)
	if err != nil {
		return nil
	}

	feeds := i.extractFeeds(b.Body.Items)

	if len(feeds) == 0 {
		return nil
	}

	for _, feed := range feeds {
		if feed.Category.Name != "" {
			dbCtg := models.Category{}
			if ctg, exists := database.CategoryWithName(feed.Category.Name, user); exists {
				dbCtg = ctg
			} else {
				dbCtg = database.NewCategory(feed.Category.Name, user)
			}

			_, err := database.NewFeedWithCategory(feed.Title, feed.Subscription, dbCtg.APIID, user)
			if err != nil {
				return err
			}
		} else {
			database.NewFeed(feed.Title, feed.Subscription, user)
		}
	}

	return nil
}
