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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo"

	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/models"
)

func (t *ServerTestSuite) TestNewTag() {
	tag := `{ "name": "Test" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(tag))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/tags")

	t.NoError(t.server.NewTag(c))
	t.Equal(http.StatusCreated, t.rec.Code)
}

func (t *ServerTestSuite) TestNewConflictingTag() {
	database.NewTag("Test", t.user)

	tag := `{ "name": "Test" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(tag))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/tags")

	t.EqualError(
		t.server.NewTag(c),
		echo.NewHTTPError(http.StatusConflict).Error(),
	)
}

func (t *ServerTestSuite) TestGetTags() {
	tag := database.NewTag("Test", t.user)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)

	c.SetPath("/v1/tags")

	t.NoError(t.server.GetTags(c))
	t.Equal(http.StatusOK, t.rec.Code)

	type Tags struct {
		Tags []models.Tag `json:"tags"`
	}

	var tags Tags
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &tags))

	t.Len(tags.Tags, 1)
	t.Equal(tag.Name, tags.Tags[0].Name)
}

func (t *ServerTestSuite) TestDeleteTag() {
	tag := database.NewTag("Test", t.user)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues(tag.APIID)

	c.SetPath("/v1/tags/:tagID")

	t.NoError(t.server.DeleteTag(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}

func (t *ServerTestSuite) TestDeleteUnknownTag() {
	req := httptest.NewRequest(echo.DELETE, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/tags/:tagID")

	t.EqualError(
		t.server.DeleteTag(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestEditTag() {
	tag := database.NewTag("test", t.user)

	mdfTagJSON := `{"name": "gopher"}`

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(mdfTagJSON))
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues(tag.APIID)

	c.SetPath("/v1/tags/:tagID")

	t.NoError(t.server.EditTag(c))
	t.Equal(http.StatusOK, t.rec.Code)
}

func (t *ServerTestSuite) TestEditUnknownTag() {
	req := httptest.NewRequest(
		echo.PUT,
		"/",
		strings.NewReader(`{"name" : "bogus" }`),
	)
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/tags/:tagID")

	t.EqualError(
		t.server.EditTag(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestTagEntries() {
	tag := database.NewTag("test", t.user)

	feed := database.NewFeed("Example", "example.com", t.user)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test",
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	req := httptest.NewRequest(
		echo.PUT,
		"/",
		strings.NewReader(fmt.Sprintf(`{
			"entries" :  ["%s"]
		}`, entry.APIID)),
	)
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues(tag.APIID)

	c.SetPath("/v1/tags/:tagID")

	t.NoError(t.server.TagEntries(c))
	t.Equal(http.StatusNoContent, t.rec.Code)
}

func (t *ServerTestSuite) TestTagEntriesWithUnknownTag() {
	req := httptest.NewRequest(
		echo.PUT,
		"/",
		strings.NewReader(`{
			"entries" :  ["foo"]
		}`),
	)
	req.Header.Set("Content-Type", "application/json")

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/tags/:tagID")

	t.EqualError(
		t.server.TagEntries(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestGetTag() {
	tag := database.NewTag("test", t.user)

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues(tag.APIID)

	c.SetPath("/v1/tags/:tagID")

	t.NoError(t.server.GetTag(c))

	var sTag models.Tag
	t.NoError(json.Unmarshal(t.rec.Body.Bytes(), &sTag))
	t.Equal(tag.Name, sTag.Name)
}

func (t *ServerTestSuite) TestGetUnknownTag() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/tags/:tagID")

	t.EqualError(
		t.server.GetTag(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (t *ServerTestSuite) TestGetEntriesFromTag() {
	tag := database.NewTag("test", t.user)

	feed := database.NewFeed("Example", "example.com", t.user)

	entry, err := database.NewEntry(models.Entry{
		Title: "Test",
	}, feed.APIID, t.user)
	t.Require().NoError(err)

	t.NoError(database.TagEntries(tag.APIID, []string{entry.APIID}, t.user))

	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues(tag.APIID)

	c.SetPath("/v1/tags/:tagID")

	t.NoError(t.server.GetEntriesFromTag(c))
	t.Equal(http.StatusOK, t.rec.Code)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	var entries Entries
	t.Require().NoError(json.Unmarshal(t.rec.Body.Bytes(), &entries))
	t.Len(entries.Entries, 1)
	t.Equal(entries.Entries[0].Title, entry.Title)
	t.Equal(entries.Entries[0].APIID, entry.APIID)
}

func (t *ServerTestSuite) TestGetEntriesFromUnknownTag() {
	req := httptest.NewRequest(echo.GET, "/", nil)

	c := t.e.NewContext(req, t.rec)
	c.Set(echoSyndUserKey, t.user)
	c.SetParamNames("tagID")
	c.SetParamValues("bogus")

	c.SetPath("/v1/tags/:tagID")

	t.EqualError(
		t.server.GetEntriesFromTag(c),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}
