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
)

//go:generate mockgen -source=exporter.go -destination=exporter_mock.go -package=services

type (
	// Exporter is an interface that wraps the basic
	// export functions.
	Exporter interface {
		Export(userID string) ([]byte, error)
	}

	// An OPMLExporter represents an exporter for the OPML 2.0 format
	// define by http://dev.opml.org/spec2.html.
	OPMLExporter struct {
		repo repo.Categories
	}
)

const (
	maxPageSize = 100
)

func NewOPMLExporter(ctgsRepo repo.Categories) OPMLExporter {
	return OPMLExporter{
		ctgsRepo,
	}
}

func marshal(feeds []models.Feed) []models.OPMLOutline {
	items := make([]models.OPMLOutline, len(feeds))

	for idx := range feeds {
		feed := feeds[idx]
		items[idx] = models.OPMLOutline{
			Title:   feed.Title,
			Text:    feed.Title,
			Type:    "rss",
			XMLUrl:  feed.Subscription,
			HTMLUrl: feed.Subscription,
		}
	}

	return items
}

// Export categories and feeds to data in OPML 2.0 format.
func (e OPMLExporter) Export(userID string) ([]byte, error) {
	var (
		continuationID string
		ctgs           []models.Category
	)

	b := models.OPML{
		Body: models.OPMLBody{},
	}

	for {
		ctgs, continuationID = e.repo.List(userID, models.Page{ContinuationID: continuationID, Count: maxPageSize})

		for idx := range ctgs {
			ctg := ctgs[idx]
			feeds, _ := e.repo.Feeds(userID, models.Page{
				FilterID:       ctg.ID,
				ContinuationID: "",
				Count:          maxPageSize,
			})
			items := marshal(feeds)
			ctgOutline := models.OPMLOutline{
				Text:  ctg.Name,
				Title: ctg.Name,
				Items: items,
			}
			b.Body.Items = append(b.Body.Items, ctgOutline)
		}

		if continuationID == "" {
			break
		}
	}

	for {
		var feeds []models.Feed
		feeds, continuationID = e.repo.Uncategorized(userID, models.Page{ContinuationID: continuationID, Count: maxPageSize})

		items := marshal(feeds)
		b.Body.Items = append(b.Body.Items, items...)

		if continuationID == "" {
			break
		}
	}

	return xml.Marshal(b)
}
