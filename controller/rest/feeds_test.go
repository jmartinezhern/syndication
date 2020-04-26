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

package rest_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	FeedsControllerSuite struct {
		suite.Suite

		ctrl      *gomock.Controller
		mockFeeds *services.MockFeeds

		controller *rest.FeedsController
		e          *echo.Echo
		user       *models.User
	}
)

func (c *FeedsControllerSuite) TestNewFeed() {
	feed := models.Feed{
		ID:           utils.CreateID(),
		Title:        "Example",
		Subscription: "https://example.com",
	}

	c.mockFeeds.EXPECT().
		New(gomock.Eq(feed.Title), gomock.Eq(feed.Subscription), gomock.Eq(""), gomock.Eq(c.user.ID)).
		Return(feed, nil)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(
		fmt.Sprintf(`{ "title": "%s", "subscription": "%s" }`, feed.Title, feed.Subscription),
	))

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/feeds")

	c.NoError(c.controller.NewFeed(ctx))
	c.Equal(http.StatusCreated, rec.Code)
}

func (c *FeedsControllerSuite) TestUnreachableNewFeed() {
	c.mockFeeds.EXPECT().
		New(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(models.Feed{}, services.ErrFetchingFeed)

	feed := `{ "title": "Example", "subscription": "bogus" }`

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(feed))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/feeds")

	c.EqualError(
		c.controller.NewFeed(ctx),
		echo.NewHTTPError(http.StatusBadRequest, "subscription url is not reachable").Error(),
	)
}

func (c *FeedsControllerSuite) TestGetFeeds() {
	page := models.Page{
		Count: 1,
	}

	c.mockFeeds.EXPECT().Feeds(gomock.Eq(c.user.ID), gomock.Eq(page)).Return([]models.Feed{{ID: utils.CreateID()}}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/feeds")

	c.NoError(c.controller.GetFeeds(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *FeedsControllerSuite) TestGetFeed() {
	feedID := utils.CreateID()

	c.mockFeeds.EXPECT().Feed(gomock.Eq(c.user.ID), gomock.Eq(feedID)).Return(models.Feed{ID: feedID}, true)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feedID)

	ctx.SetPath("/v1/feeds/:feedID")

	c.NoError(c.controller.GetFeed(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *FeedsControllerSuite) TestGetUnknownFeed() {
	c.mockFeeds.EXPECT().Feed(gomock.Any(), gomock.Any()).Return(models.Feed{}, false)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID")

	c.EqualError(
		c.controller.GetFeed(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) TestEditFeed() {
	feedID := utils.CreateID()

	c.mockFeeds.EXPECT().
		Update(gomock.Eq(c.user.ID), gomock.Eq(&models.Feed{ID: feedID, Title: "NewName"})).
		Return(nil)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{ "title": "NewName" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feedID)

	ctx.SetPath("/v1/feeds/:feedID")

	c.NoError(c.controller.EditFeed(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *FeedsControllerSuite) TestEditUnknownFeed() {
	c.mockFeeds.EXPECT().Update(gomock.Any(), gomock.Any()).Return(services.ErrFeedNotFound)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{ "title": "title" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID")

	c.EqualError(
		c.controller.EditFeed(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) TestDeleteFeed() {
	feedID := utils.CreateID()

	c.mockFeeds.EXPECT().Delete(gomock.Eq(c.user.ID), gomock.Eq(feedID)).Return(nil)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feedID)

	ctx.SetPath("/v1/feeds/:feedID")

	c.NoError(c.controller.DeleteFeed(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *FeedsControllerSuite) TestDeleteUnknownFeed() {
	c.mockFeeds.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(services.ErrFeedNotFound)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.DeleteFeed(ctx).Error(),
	)
}

func (c *FeedsControllerSuite) TestMarkFeed() {
	feedID := utils.CreateID()

	c.mockFeeds.EXPECT().
		Mark(gomock.Eq(c.user.ID), gomock.Eq(feedID), gomock.Eq(models.MarkerRead)).
		Return(nil)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feedID)

	ctx.SetPath("/v1/feeds/:feedID/mark")

	c.NoError(c.controller.MarkFeed(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *FeedsControllerSuite) TestMarkUnknownFeeed() {
	c.mockFeeds.EXPECT().Mark(gomock.Any(), gomock.Any(), gomock.Any()).Return(services.ErrFeedNotFound)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID/mark")

	c.EqualError(
		c.controller.MarkFeed(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) TestGetFeedEntries() {
	feedID := utils.CreateID()

	page := models.Page{
		Count:    1,
		FilterID: feedID,
		Newest:   true,
		Marker:   models.MarkerAny,
	}

	c.mockFeeds.EXPECT().
		Entries(gomock.Eq(c.user.ID), gomock.Eq(page)).
		Return([]models.Entry{{ID: utils.CreateID()}}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feedID)
	ctx.SetPath("/v1/feeds/:feedID/entries")

	c.NoError(c.controller.GetFeedEntries(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *FeedsControllerSuite) TestGetFeedStats() {
	feedID := utils.CreateID()

	c.mockFeeds.EXPECT().Stats(gomock.Eq(c.user.ID), gomock.Eq(feedID)).Return(models.Stats{}, nil)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues(feedID)

	ctx.SetPath("/v1/feeds/:feedID/stats")

	c.NoError(c.controller.GetFeedStats(ctx))

	var stats models.Stats

	c.NoError(json.Unmarshal(rec.Body.Bytes(), &stats))
}

func (c *FeedsControllerSuite) TestGetUnknownFeedStats() {
	c.mockFeeds.EXPECT().Stats(gomock.Any(), gomock.Any()).Return(models.Stats{}, services.ErrFeedNotFound)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("feedID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/feeds/:feedID/stats")

	c.EqualError(
		c.controller.GetFeedStats(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *FeedsControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.mockFeeds = services.NewMockFeeds(c.ctrl)

	c.controller = rest.NewFeedsController(c.mockFeeds, c.e)
}

func (c *FeedsControllerSuite) TearDownTest() {
	c.ctrl.Finish()
}

func TestFeedsControllerSuite(t *testing.T) {
	suite.Run(t, new(FeedsControllerSuite))
}
