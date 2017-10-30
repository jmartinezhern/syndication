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

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	//"github.com/stretchr/testify/suite"
	//"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/sync"
)

const TestDBPath = "/tmp/syndication-test-server.db"

type (
	ServerTestSuite struct {
		suite.Suite

		db     *database.DB
		sync   *sync.Sync
		server *Server
		user   models.User
		token  string
		ts     *httptest.Server
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (suite *ServerTestSuite) SetupTest() {
	randUserName := RandStringRunes(8)
	fmt.Println(randUserName)

	resp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Require().Equal(204, resp.StatusCode)

	err = resp.Body.Close()
	suite.Nil(err)

	resp, err = http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(resp.StatusCode, 200)

	type Token struct {
		Token string `json:"token"`
	}

	var t Token
	err = json.NewDecoder(resp.Body).Decode(&t)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(t.Token)

	suite.token = t.Token

	suite.user, err = suite.db.UserWithName(randUserName)
	suite.Require().Nil(err)

	err = resp.Body.Close()
	suite.Nil(err)
}

func (suite *ServerTestSuite) TearDownTest() {
	suite.db.DeleteUser(suite.user.APIID)
}

func (suite *ServerTestSuite) TestRequestWithNonJSONType() {
	payload := []byte(`{"title":"RSS Test", "subscription": "https://www.eff.org/rss/updates.xml"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/feeds", bytes.NewBuffer(payload))

	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(400, resp.StatusCode)
}

func (suite *ServerTestSuite) TestNewFeed() {
	payload := []byte(`{"title":"RSS Test", "subscription": "` + suite.ts.URL + `"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/feeds", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respFeed.APIID)
	suite.NotEmpty(respFeed.Title)

	dbFeed, err := suite.db.Feed(respFeed.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(dbFeed.Title, respFeed.Title)
}

func (suite *ServerTestSuite) TestNewUnretrivableFeed() {
	payload := []byte(`{"title":"EFF", "subscription": "https://localhost:17170/rss/updates.xml"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/feeds", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(400, resp.StatusCode)
}

func (suite *ServerTestSuite) TestGetFeeds() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			Title:        "Feed " + strconv.Itoa(i+1),
			Subscription: "http://example.com/feed",
		}
		err := suite.db.NewFeed(&feed, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(feed.ID)
		suite.Require().NotEmpty(feed.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	suite.Require().Nil(err)
	suite.Len(respFeeds.Feeds, 5)
}

func (suite *ServerTestSuite) TestGetFeed() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds/"+feed.APIID, nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	suite.Require().Nil(err)

	suite.Equal(respFeed.Title, feed.Title)
	suite.Equal(respFeed.APIID, feed.APIID)
}

func (suite *ServerTestSuite) TestEditFeed() {
	feed := models.Feed{Subscription: suite.ts.URL}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	payload := []byte(`{"title": "EFF Updates"}`)
	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/feeds/"+feed.APIID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	respFeed, err := suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)
	suite.Equal(respFeed.Title, "EFF Updates")
}

func (suite *ServerTestSuite) TestDeleteFeed() {
	feed := models.Feed{Subscription: suite.ts.URL}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	req, err := http.NewRequest("DELETE", "http://localhost:9876/v1/feeds/"+feed.APIID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Feed(feed.APIID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestGetEntriesFromFeed() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds/"+feed.APIID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestMarkFeed() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/feeds/"+feed.APIID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)
}

func (suite *ServerTestSuite) TestNewCategory() {
	payload := []byte(`{"name": "News"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/categories", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respCtg.APIID)
	suite.NotEmpty(respCtg.Name)

	dbCtg, err := suite.db.Category(respCtg.APIID, &suite.user)
	suite.Nil(err)
	suite.Equal(dbCtg.Name, respCtg.Name)
}

func (suite *ServerTestSuite) TestNewTag() {
	payload := []byte(`{"name": "News"}`)
	req, err := http.NewRequest("POST", "http://localhost:9876/v1/tags", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respTag := new(models.Tag)
	err = json.NewDecoder(resp.Body).Decode(respTag)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respTag.APIID)
	suite.NotEmpty(respTag.Name)

	dbTag, err := suite.db.Tag(respTag.APIID, &suite.user)
	suite.Nil(err)
	suite.Equal(dbTag.Name, respTag.Name)
}

func (suite *ServerTestSuite) TestGetCategories() {
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			Name: "Category " + strconv.Itoa(i+1),
		}
		err := suite.db.NewCategory(&ctg, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(ctg.ID)
		suite.Require().NotEmpty(ctg.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Categories struct {
		Categories []models.Category `json:"categories"`
	}

	respCtgs := new(Categories)
	err = json.NewDecoder(resp.Body).Decode(respCtgs)
	suite.Require().Nil(err)

	suite.Len(respCtgs.Categories, 7)
}

func (suite *ServerTestSuite) TestGetTags() {
	for i := 0; i < 5; i++ {
		tag := models.Tag{
			Name: "Tag " + strconv.Itoa(i+1),
		}
		err := suite.db.NewTag(&tag, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(tag.ID)
		suite.Require().NotEmpty(tag.APIID)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/tags", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	respTags := new(Tags)
	err = json.NewDecoder(resp.Body).Decode(respTags)
	suite.Require().Nil(err)

	suite.Len(respTags.Tags, 5)
}

func (suite *ServerTestSuite) TestGetCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+ctg.APIID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	suite.Require().Nil(err)

	suite.Equal(respCtg.Name, ctg.Name)
	suite.Equal(respCtg.APIID, ctg.APIID)
}

func (suite *ServerTestSuite) TestGetTag() {
	tag := models.Tag{Name: "News"}
	err := suite.db.NewTag(&tag, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(tag.ID)
	suite.Require().NotEmpty(tag.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/tags/"+tag.APIID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respTag := new(models.Tag)
	err = json.NewDecoder(resp.Body).Decode(respTag)
	suite.Require().Nil(err)

	suite.Equal(respTag.Name, tag.Name)
	suite.Equal(respTag.APIID, tag.APIID)
}

func (suite *ServerTestSuite) TestEditCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.APIID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/categories/"+ctg.APIID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	editedCtg, err := suite.db.Category(ctg.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(editedCtg.Name, "World News")
}

func (suite *ServerTestSuite) TestEditTag() {
	tag := models.Tag{Name: "News"}
	err := suite.db.NewTag(&tag, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(tag.ID)
	suite.Require().NotEmpty(tag.APIID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/tags/"+tag.APIID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	editedTag, err := suite.db.Tag(tag.APIID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(editedTag.Name, "World News")
}

func (suite *ServerTestSuite) TestDeleteCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.APIID)

	req, err := http.NewRequest("DELETE", "http://localhost:9876/v1/categories/"+ctg.APIID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Category(ctg.APIID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestDeleteTag() {
	tag := models.Tag{Name: "News"}
	err := suite.db.NewTag(&tag, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(tag.ID)
	suite.Require().NotEmpty(tag.APIID)

	req, err := http.NewRequest("DELETE", "http://localhost:9876/v1/tags/"+tag.APIID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Category(tag.APIID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestGetFeedsFromCategory() {
	category := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.APIID)

	feed := models.Feed{
		Title:        "Test feed",
		Subscription: "http://localhost:9876",
		Category:     category,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+category.APIID+"/feeds", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	suite.Require().Nil(err)
	suite.Len(respFeeds.Feeds, 1)
}

func (suite *ServerTestSuite) TestGetEntriesFromCategory() {
	category := models.Category{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.APIID)
	suite.Require().NotZero(category.ID)

	feed := models.Feed{
		Subscription: suite.ts.URL,
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+category.APIID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestGetEntriesFromTag() {
	tag := models.Tag{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewTag(&tag, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(tag.APIID)
	suite.Require().NotZero(tag.ID)

	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(entries)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	err = suite.db.TagEntries(tag.APIID, entryAPIIDs, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/tags/"+tag.APIID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestTagEntries() {
	tag := models.Tag{
		Name: "News",
	}
	err := suite.db.NewTag(&tag, &suite.user)
	suite.Require().Nil(err)

	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	type EntryIds struct {
		Entries []string `json:"entries"`
	}

	var entries []models.Entry
	for i := 0; i < 5; i++ {
		entry := models.Entry{
			Title:  "Test Entry",
			Author: "varddum",
			Link:   "http://example.com",
			Mark:   models.Unread,
			Feed:   feed,
		}

		entries = append(entries, entry)
	}

	err = suite.db.NewEntries(entries, &feed, &suite.user)
	suite.Require().Nil(err)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Any, &suite.user)
	suite.Require().Nil(err)

	entryAPIIDs := make([]string, len(entries))
	for i, entry := range entries {
		entryAPIIDs[i] = entry.APIID
	}

	list := EntryIds{
		Entries: entryAPIIDs,
	}

	b, err := json.Marshal(list)
	suite.Require().Nil(err)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/tags/"+tag.APIID+"/entries", bytes.NewBuffer(b))
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNoContent, resp.StatusCode)

	taggedEntries, err := suite.db.EntriesFromTag(tag.APIID, models.Any, true, &suite.user)
	suite.Nil(err)
	suite.Len(taggedEntries, 5)
}

func (suite *ServerTestSuite) TestMarkCategory() {
	category := models.Category{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.APIID)

	feed := models.Feed{
		Subscription: suite.ts.URL,
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromCategory(category.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	entries, err = suite.db.EntriesFromCategory(category.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/categories/"+category.APIID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromCategory(category.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromCategory(category.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)
}

func (suite *ServerTestSuite) TestGetEntries() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestGetEntry() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	entry := models.Entry{
		Title:  "Item 1",
		Link:   "http://localhost:9876/item_1",
		Feed:   feed,
		FeedID: feed.ID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(entry.ID)
	suite.Require().NotEmpty(entry.APIID)

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries/"+entry.APIID, nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respEntry := new(models.Entry)
	err = json.NewDecoder(resp.Body).Decode(respEntry)
	suite.Require().Nil(err)

	suite.Equal(entry.Title, respEntry.Title)
	suite.Equal(entry.APIID, respEntry.APIID)
}

func (suite *ServerTestSuite) TestMarkEntry() {
	feed := models.Feed{
		Subscription: suite.ts.URL,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.APIID)

	err = suite.server.sync.SyncFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/entries/"+entries[0].APIID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 4)

	entries, err = suite.db.EntriesFromFeed(feed.APIID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)
}

func (suite *ServerTestSuite) TestGetStatsForFeed() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Read,
			Saved:  true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/feeds/"+feed.APIID+"/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestGetStats() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Read,
			Saved:  true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/entries/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestGetStatsForCategory() {
	category := models.Category{
		Name: "World",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.APIID)

	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
		Category:     category,
		CategoryID:   category.ID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.APIID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Read,
			Saved:  true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:  "Item",
			Link:   "http://example.com",
			Feed:   feed,
			FeedID: feed.ID,
			Mark:   models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	req, err := http.NewRequest("GET", "http://localhost:9876/v1/categories/"+category.APIID+"/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestAddFeedsToCategory() {
	feed := models.Feed{
		Title:        "Example Feed",
		Subscription: "http://example.com/feed",
	}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	ctg := models.Category{
		Name: "Test",
	}
	err = suite.db.NewCategory(&ctg, &suite.user)
	suite.Nil(err)

	type FeedList struct {
		Feeds []string `json:"feeds"`
	}

	list := FeedList{
		Feeds: []string{feed.APIID},
	}

	b, err := json.Marshal(list)
	suite.Require().Nil(err)

	req, err := http.NewRequest("PUT", "http://localhost:9876/v1/categories/"+ctg.APIID+"/feeds", bytes.NewBuffer(b))
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNoContent, resp.StatusCode)

	feed, err = suite.db.Feed(feed.APIID, &suite.user)
	suite.Nil(err)

	suite.Equal(ctg.ID, feed.CategoryID)
}

func (suite *ServerTestSuite) TestRegister() {
	suite.db.DeleteUser(suite.user.APIID)

	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(204, regResp.StatusCode)

	users := suite.db.Users("username")
	suite.Len(users, 1)

	suite.Equal(randUserName, users[0].Username)
	suite.NotEmpty(users[0].ID)
	suite.NotEmpty(users[0].APIID)

	err = regResp.Body.Close()
	suite.Nil(err)

	suite.db.DeleteUser(users[0].APIID)
}

func (suite *ServerTestSuite) TestLogin() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(204, regResp.StatusCode)

	err = regResp.Body.Close()
	suite.Require().Nil(err)

	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(200, loginResp.StatusCode)

	type Token struct {
		Token string `json:"token"`
	}

	var token Token
	err = json.NewDecoder(loginResp.Body).Decode(&token)
	suite.Require().Nil(err)
	suite.NotEmpty(token.Token)

	user, err := suite.db.UserWithName(randUserName)
	suite.Nil(err)

	err = loginResp.Body.Close()
	suite.Nil(err)

	suite.db.DeleteUser(user.APIID)
}

func (suite *ServerTestSuite) TestLoginWithNonExistentUser() {
	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {"bogus"}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	suite.Equal(401, loginResp.StatusCode)

	err = loginResp.Body.Close()
	suite.Nil(err)
}

func (suite *ServerTestSuite) TestLoginWithBadPassword() {
	randUserName := RandStringRunes(8)
	regResp, err := http.PostForm("http://localhost:9876/v1/register",
		url.Values{"username": {randUserName}, "password": {"testtesttest"}})
	suite.Require().Nil(err)

	user, err := suite.db.UserWithName(randUserName)
	suite.Require().Nil(err)
	defer suite.db.DeleteUser(user.APIID)

	suite.Equal(204, regResp.StatusCode)

	err = regResp.Body.Close()
	suite.Require().Nil(err)

	loginResp, err := http.PostForm("http://localhost:9876/v1/login",
		url.Values{"username": {randUserName}, "password": {"bogus"}})
	suite.Require().Nil(err)

	suite.Equal(401, loginResp.StatusCode)

	err = loginResp.Body.Close()
	suite.Nil(err)
}

func (suite *ServerTestSuite) startServer() {
	conf := config.DefaultConfig
	conf.Server.HTTPPort = 9876
	conf.Server.AuthSecret = "secret"

	var err error
	suite.db, err = database.NewDB(config.Database{
		Type:       "sqlite3",
		Connection: TestDBPath,
		APIKeyExpiration: config.Duration{
			Duration: time.Hour * 72,
		},
	})
	suite.Require().Nil(err)

	suite.sync = sync.NewSync(suite.db, config.Sync{
		SyncInterval: config.Duration{Duration: time.Second * 5},
	})

	if suite.server == nil {
		suite.server = NewServer(suite.db, suite.sync, conf.Server)
		suite.server.handle.HideBanner = true
		go suite.server.Start()
	}

	time.Sleep(time.Second)

	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:9876</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:9876/item_1</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:9876/item_2</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:9876/item_3</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:9876/item_4</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:9876/item_5</link>
      <description>Single test item</description>
      <author>varddum</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:9876/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	suite.ts = httptest.NewServer(http.HandlerFunc(handler))
}

func TestServerTestSuite(t *testing.T) {
	serverSuite := new(ServerTestSuite)
	serverSuite.startServer()
	suite.Run(t, serverSuite)
	serverSuite.server.Stop()
	os.Remove(TestDBPath)
	serverSuite.ts.Close()
}
