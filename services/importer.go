/*
 *   Copyright (C) 2021. Jorge Martinez Hernandez
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU Affero General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU Affero General Public License for more details.
 *
 *   You should have received a copy of the GNU Affero General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package services

import (
	"encoding/xml"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

//go:generate mockgen -source=importer.go -destination=importer_mock.go -package=services

type (
	// Importer is an interface that wraps the basic
	// import functions.
	Importer interface {
		Import([]byte, string) error
	}

	// An OPMLImporter represents an importer for the OPML 2.0 format
	// define by http://dev.opml.org/spec2.html.
	OPMLImporter struct {
		ctgsRepo  repo.Categories
		feedsRepo repo.Feeds
	}
)

func NewOPMLImporter(ctgsRepo repo.Categories, feedsRepo repo.Feeds) OPMLImporter {
	return OPMLImporter{
		ctgsRepo,
		feedsRepo,
	}
}

func (i OPMLImporter) extractFeeds(items []models.OPMLOutline) []models.Feed {
	var feeds []models.Feed

	for idx := range items {
		outline := items[idx]
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
			for idx := range outline.Items {
				ctgOutline := outline.Items[idx]
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
func (i OPMLImporter) Import(data []byte, userID string) error {
	b := models.OPML{}

	err := xml.Unmarshal(data, &b)
	if err != nil {
		return nil
	}

	feeds := i.extractFeeds(b.Body.Items)

	if len(feeds) == 0 {
		return nil
	}

	for idx := range feeds {
		feed := feeds[idx]
		if feed.Category.Name != "" {
			dbCtg := models.Category{}
			if ctg, exists := i.ctgsRepo.CategoryWithName(userID, feed.Category.Name); exists {
				dbCtg = ctg
			} else {
				dbCtg = models.Category{
					ID:   utils.CreateID(),
					Name: feed.Category.Name,
				}
				i.ctgsRepo.Create(userID, &dbCtg)
			}

			feed.ID = utils.CreateID()
			feed.Category = dbCtg
			i.feedsRepo.Create(userID, &feed)
		} else {
			feed.ID = utils.CreateID()
			i.feedsRepo.Create(userID, &feed)
		}
	}

	return nil
}
