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
	// Importer is an interface that wraps the basic
	// import functions.
	Importer interface {
		Import([]byte) []Feed
	}

	// An OPMLImporter represents an importer for the OPML 2.0 format
	// define by http://dev.opml.org/spec2.html.
	OPMLImporter struct{}
)

// NewOPMLImporter creates a new OPMLImporter instance
func NewOPMLImporter() OPMLImporter {
	return OPMLImporter{}
}

// Import data that must be in a OPML 2.0 format.
func (i OPMLImporter) Import(data []byte) []Feed {
	b := OPML{}

	err := xml.Unmarshal(data, &b)
	if err != nil {
		return nil
	}

	feeds := []Feed{}

	for _, outline := range b.Body.Items {
		if outline.Type == "rss" {
			feed := Feed{
				Title:        outline.Title,
				Subscription: outline.XMLUrl,
			}

			feeds = append(feeds, feed)
		} else if outline.Type == "" && len(outline.Items) > 0 {
			// We consider this a category
			ctg := Category{
				Name: outline.Title,
			}
			for _, ctgOutline := range outline.Items {
				feed := Feed{
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
