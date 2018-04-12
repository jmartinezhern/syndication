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

package models

import (
	"encoding/xml"
)

type (
	// FeedExporter is an interface that wraps the basic
	// export functions.
	FeedExporter interface {
		Export([]Category) ([]byte, error)
	}

	// An OPMLExporter represents an exporter for the OPML 2.0 format
	// define by http://dev.opml.org/spec2.html.
	OPMLExporter struct{}
)

// NewOPMLExporter creates a new instance of OPMLExporter
func NewOPMLExporter() OPMLExporter {
	return OPMLExporter{}
}

// Export categories and feeds to data in OPML 2.0 format.
func (i OPMLExporter) Export(ctgs []Category) ([]byte, error) {
	b := OPML{
		Body: OPMLBody{},
	}

	for _, ctg := range ctgs {
		if ctg.Name != Uncategorized {
			ctgOutline := OPMLOutline{
				Text:  ctg.Name,
				Title: ctg.Name,
			}

			for _, feed := range ctg.Feeds {
				outline := OPMLOutline{
					Title:   feed.Title,
					Text:    feed.Title,
					Type:    rssType,
					XMLUrl:  feed.Subscription,
					HTMLUrl: feed.Subscription,
				}

				ctgOutline.Items = append(ctgOutline.Items, outline)
			}

			b.Body.Items = append(b.Body.Items, ctgOutline)
		} else {
			for _, feed := range ctg.Feeds {
				outline := OPMLOutline{
					Title:   feed.Title,
					Text:    feed.Title,
					Type:    rssType,
					XMLUrl:  feed.Subscription,
					HTMLUrl: feed.Subscription,
				}

				b.Body.Items = append(b.Body.Items, outline)
			}
		}
	}

	return xml.Marshal(b)
}
