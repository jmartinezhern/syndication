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
	"encoding/xml"

	"github.com/varddum/syndication/models"
)

const (
	RSSType = "rss"
)

type (
	FeedImporter interface {
		Import([]byte) []models.Feed
		Export([]models.Category) ([]byte, error)
	}

	OPMLImporter struct {
	}

	OPMLOutline struct {
		XMLName xml.Name      `xml:"outline"`
		Type    string        `xml:"type,attr"`
		Text    string        `xml:"text,attr"`
		Title   string        `xml:"title,attr"`
		HTMLUrl string        `xml:"htmlUrl,attr"`
		XMLUrl  string        `xml:"xmlUrl,attr"`
		Items   []OPMLOutline `xml:"outline"`
	}

	OPMLBody struct {
		XMLName xml.Name      `xml:"body"`
		Items   []OPMLOutline `xml:"outline"`
	}

	OPML struct {
		XMLName xml.Name `xml:"opml"`
		Body    OPMLBody `xml:"body"`
	}
)

func NewOPMLImporter() OPMLImporter {
	return OPMLImporter{}
}

func (i OPMLImporter) Import(data []byte) []models.Feed {
	b := OPML{}

	err := xml.Unmarshal(data, &b)
	if err != nil {
		return nil
	}

	feeds := []models.Feed{}

	for _, outline := range b.Body.Items {
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

func (i OPMLImporter) Export(ctgs []models.Category) ([]byte, error) {
	b := OPML{
		Body: OPMLBody{},
	}

	for _, ctg := range ctgs {
		if ctg.Name != models.Uncategorized {
			ctgOutline := OPMLOutline{
				Text:  ctg.Name,
				Title: ctg.Name,
			}

			for _, feed := range ctg.Feeds {
				outline := OPMLOutline{
					Title:   feed.Title,
					Text:    feed.Title,
					Type:    RSSType,
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
					Type:    RSSType,
					XMLUrl:  feed.Subscription,
					HTMLUrl: feed.Subscription,
				}

				b.Body.Items = append(b.Body.Items, outline)
			}
		}
	}

	return xml.Marshal(b)
}
