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

package database

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/utils"
)

type TagsSuite struct {
	suite.Suite

	user models.User
	ctg  models.Category
	feed models.Feed
}

func (s *TagsSuite) TestNewTag() {
	tagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: tagID,
		Name:  "tech",
	}, s.user)

	tag, found := TagWithAPIID(tagID, s.user)
	s.True(found)
	s.Equal("tech", tag.Name)
}

func (s *TagsSuite) TestTagEntries() {
	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:  utils.CreateAPIID(),
			Title:  "Test Entry",
			Author: "John Doe",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		}

		entries = append(entries, entry)
	}

	entries, err := NewEntries(entries, s.feed.APIID, s.user)
	s.Require().NotEmpty(entries)
	s.Require().NoError(err)

	entries = FeedEntries(s.feed.APIID, true, models.MarkerAny, s.user)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	tagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: tagID,
		Name:  "tech",
	}, s.user)

	tag, found := TagWithAPIID(tagID, s.user)
	s.True(found)
	s.Equal("tech", tag.Name)

	err = TagEntries(tag.APIID, entryAPIIDs, s.user)
	s.NoError(err)

	taggedEntries := EntriesFromTag(tag.APIID, models.MarkerAny, true, s.user)
	s.Len(taggedEntries, 5)
}

func (s *TagsSuite) TestTagUnknownEntries() {
	err := TagEntries("bogus", make([]string, 1), s.user)
	s.Equal(ErrModelNotFound, err)
}

func (s *TagsSuite) TestTagNoEntries() {
	err := TagEntries("bogus", nil, s.user)
	s.NoError(err)
}

func (s *TagsSuite) TestTagMultipleEntries() {
	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			APIID:  utils.CreateAPIID(),
			Title:  "Test Entry " + strconv.Itoa(i),
			Author: "John Doe",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		}

		entries = append(entries, entry)
	}

	_, err := NewEntries(entries, s.feed.APIID, s.user)
	s.Require().NoError(err)

	firstTagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: firstTagID,
		Name:  "first",
	}, s.user)

	secondTagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: secondTagID,
		Name:  "second",
	}, s.user)

	entries = FeedEntries(s.feed.APIID, true, models.MarkerAny, s.user)
	s.Require().NoError(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = TagEntries(firstTagID, entryAPIIDs, s.user)
	s.NoError(err)

	err = TagEntries(secondTagID, entryAPIIDs, s.user)
	s.NoError(err)

	taggedEntries := EntriesFromTag(firstTagID, models.MarkerAny, true, s.user)
	s.Len(taggedEntries, 5)

	taggedEntries = EntriesFromTag(secondTagID, models.MarkerAny, true, s.user)
	s.Len(taggedEntries, 5)
}

func (s *TagsSuite) TestEntriesFromMultipleTags() {
	entries := []models.Entry{
		{
			APIID:  utils.CreateAPIID(),
			Title:  "Test Entry",
			Author: "John Doe",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		},
		{
			APIID:  utils.CreateAPIID(),
			Title:  "Test Entry",
			Author: "John Doe",
			Link:   "http://example.com",
			Mark:   models.MarkerUnread,
		},
	}

	entries, err := NewEntries(entries, s.feed.APIID, s.user)
	s.Require().NoError(err)

	firstTagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: firstTagID,
		Name:  "first",
	}, s.user)

	secondTagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: secondTagID,
		Name:  "second",
	}, s.user)

	s.NoError(TagEntries(firstTagID, []string{entries[0].APIID}, s.user))
	s.NoError(TagEntries(secondTagID, []string{entries[1].APIID}, s.user))

	taggedEntries := EntriesFromMultipleTags([]string{firstTagID, secondTagID}, true, models.MarkerAny, s.user)
	s.Len(taggedEntries, 2)
}

func (s *TagsSuite) TestDeleteTag() {
	tagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: tagID,
		Name:  "news",
	}, s.user)

	err := DeleteTag(tagID, s.user)
	s.NoError(err)

	_, found := TagWithAPIID(tagID, s.user)
	s.False(found)
}

func (s *TagsSuite) TestEditTag() {
	tagID := utils.CreateAPIID()
	CreateTag(&models.Tag{
		APIID: tagID,
		Name:  "news",
	}, s.user)

	tag, err := EditTag(tagID, models.Tag{Name: "World News"}, s.user)
	s.NoError(err)
	s.Equal("World News", tag.Name)
}

func (s *TagsSuite) SetupTest() {
	err := Init("sqlite3", ":memory:")

	s.Require().NoError(err)

	s.user = models.User{
		APIID:    utils.CreateAPIID(),
		Username: "test",
	}

	CreateUser(&s.user)
	s.ctg = models.Category{
		APIID: utils.CreateAPIID(),
		Name:  models.Uncategorized,
	}
	CreateCategory(&s.ctg, s.user)

	s.feed = models.Feed{
		APIID:        utils.CreateAPIID(),
		Title:        "Test site",
		Subscription: "http://example.com",
	}
	err = CreateFeed(&s.feed, s.ctg.APIID, s.user)
	s.Require().NoError(err)
}

func (s *TagsSuite) TearDownTest() {
	err := Close()
	s.NoError(err)
}

func TestTagsSuite(t *testing.T) {
	suite.Run(t, new(TagsSuite))
}
