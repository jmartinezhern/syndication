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
	"errors"
	"github.com/golang/mock/gomock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/utils"
)

const (
	userContextKey = "user"
)

type (
	CategoriesControllerSuite struct {
		suite.Suite

		ctrl           *gomock.Controller
		mockCategories *services.MockCategories

		controller *rest.CategoriesController
		e          *echo.Echo
		user       *models.User
	}
)

func (c *CategoriesControllerSuite) TestNewCategory() {
	ctg := models.Category{
		ID: utils.CreateID(),
	}

	c.mockCategories.EXPECT().New(gomock.Eq(c.user.ID), gomock.Eq("new")).Return(ctg, nil)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`{ "name": "new" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.NoError(c.controller.NewCategory(ctx))
	c.Equal(http.StatusCreated, rec.Code)

	response := &models.Category{}

	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), response))

	c.Equal(ctg.ID, response.ID)
}

func (c *CategoriesControllerSuite) TestNewConflictingCategory() {
	c.mockCategories.EXPECT().
		New(gomock.Eq(c.user.ID), gomock.Eq("test")).
		Return(models.Category{}, services.ErrCategoryConflicts)

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`{ "name": "test" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	err := c.controller.NewCategory(ctx)

	c.EqualError(
		err,
		echo.NewHTTPError(http.StatusConflict).Error(),
	)
}

func (c *CategoriesControllerSuite) TestNewCategoryWithBadInput() {
	req := httptest.NewRequest(echo.POST, "/", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.EqualError(
		c.controller.NewCategory(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *CategoriesControllerSuite) TestNewCategoryInternalError() {
	c.mockCategories.EXPECT().
		New(gomock.Eq(c.user.ID), gomock.Eq("test")).
		Return(models.Category{}, errors.New("error"))

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(`{ "name": "test" }`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	err := c.controller.NewCategory(ctx)

	c.EqualError(
		err,
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}

	c.mockCategories.EXPECT().
		Category(gomock.Eq(c.user.ID), gomock.Eq(ctg.ID)).
		Return(ctg, true)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)

	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories/:categoryID")
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	c.NoError(c.controller.GetCategory(ctx))
	c.Equal(http.StatusOK, rec.Code)

	response := &models.Category{}

	c.NoError(json.Unmarshal(rec.Body.Bytes(), response))
	c.Equal(ctg.Name, response.Name)
}

func (c *CategoriesControllerSuite) TestGetMissingCategory() {
	c.mockCategories.EXPECT().Category(gomock.Eq(c.user.ID), gomock.Any()).Return(models.Category{}, false)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories/:categoryID")
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.GetCategory(ctx).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategories() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "test",
	}

	c.mockCategories.EXPECT().
		Categories(gomock.Eq(c.user.ID), gomock.Eq(models.Page{ContinuationID: "", Count: 1})).
		Return([]models.Category{ctg}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.NoError(c.controller.GetCategories(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type ctgs struct {
		Categories     []models.Category `json:"categories"`
		ContinuationID string            `json:"continuationID"`
	}

	var categories ctgs

	c.NoError(json.Unmarshal(rec.Body.Bytes(), &categories))
	c.Require().Len(categories.Categories, 1)
	c.Equal(ctg.Name, categories.Categories[0].Name)
}

func (c *CategoriesControllerSuite) TestGetCategoriesBadRequest() {
	req := httptest.NewRequest(echo.GET, "/?count=true", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)

	ctx.SetPath("/v1/categories")

	c.EqualError(
		c.controller.GetCategories(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryFeeds() {
	ctg := models.Category{ID: utils.CreateID()}

	feed := models.Feed{
		ID:           utils.CreateID(),
		Subscription: "http://localhost:9090",
		Title:        "example",
		Category:     ctg,
	}

	c.mockCategories.EXPECT().Feeds(gomock.Eq(c.user.ID), gomock.Any()).Return([]models.Feed{feed}, "")

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID/feeds")

	c.NoError(c.controller.GetCategoryFeeds(ctx))
	c.Equal(http.StatusOK, rec.Code)

	type feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	var ctgFeeds feeds

	c.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &ctgFeeds))

	c.Len(ctgFeeds.Feeds, 1)
}

func (c *CategoriesControllerSuite) TestGetCategoryFeedsBadRequest() {
	req := httptest.NewRequest(echo.GET, "/?count=true", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID/feeds")

	c.EqualError(
		c.controller.GetCategoryFeeds(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *CategoriesControllerSuite) TestEditCategory() {
	ctg := models.Category{
		ID:   utils.CreateID(),
		Name: "gopher",
	}

	c.mockCategories.EXPECT().Update(gomock.Eq(c.user.ID), gomock.Eq(ctg.ID), gomock.Eq(ctg.Name)).Return(ctg, nil)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name": "gopher"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctg.ID)

	ctx.SetPath("/v1/categories/:categoryID")

	c.NoError(c.controller.EditCategory(ctx))
	c.Equal(http.StatusOK, rec.Code)
}

func (c *CategoriesControllerSuite) TestEditCategoryBadRequest() {
	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID")

	c.EqualError(
		c.controller.EditCategory(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *CategoriesControllerSuite) TestEditCategoryNotFound() {
	c.mockCategories.EXPECT().
		Update(gomock.Eq(c.user.ID), gomock.Any(), gomock.Any()).
		Return(models.Category{}, services.ErrCategoryNotFound)

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name": "gopher"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID")

	c.EqualError(
		c.controller.EditCategory(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *CategoriesControllerSuite) TestEditCategoryInternalError() {
	c.mockCategories.EXPECT().
		Update(gomock.Eq(c.user.ID), gomock.Any(), gomock.Any()).
		Return(models.Category{}, errors.New("error"))

	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{"name": "gopher"}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID")

	c.EqualError(
		c.controller.EditCategory(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *CategoriesControllerSuite) TestAppendFeeds() {
	ctgID := utils.CreateID()

	c.mockCategories.EXPECT().AddFeeds(gomock.Eq(c.user.ID), gomock.Eq(ctgID), gomock.Any())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{ "feeds": ["id"] }`))
	req.Header.Set("Content-Type", "application/json")

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctgID)

	ctx.SetPath("/v1/categories/:categoryID/feeds")

	c.NoError(c.controller.AppendCategoryFeeds(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *CategoriesControllerSuite) TestAppendFeedsBadRequest() {
	req := httptest.NewRequest(echo.PUT, "/", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID/feeds")

	c.EqualError(
		c.controller.AppendCategoryFeeds(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *CategoriesControllerSuite) TestDeleteCategory() {
	ctgID := utils.CreateID()

	c.mockCategories.EXPECT().Delete(gomock.Eq(c.user.ID), gomock.Eq(ctgID)).Return(nil)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctgID)

	ctx.SetPath("/v1/categories/:categoryID")

	c.NoError(c.controller.DeleteCategory(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *CategoriesControllerSuite) TestDeleteCategoryNotFound() {
	c.mockCategories.EXPECT().Delete(gomock.Eq(c.user.ID), gomock.Any()).Return(services.ErrCategoryNotFound)

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID")

	c.EqualError(
		c.controller.DeleteCategory(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *CategoriesControllerSuite) TestDeleteCategoryInternalError() {
	c.mockCategories.EXPECT().Delete(gomock.Eq(c.user.ID), gomock.Any()).Return(errors.New("error"))

	req := httptest.NewRequest(echo.DELETE, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("id")

	ctx.SetPath("/v1/categories/:categoryID")

	c.EqualError(
		c.controller.DeleteCategory(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *CategoriesControllerSuite) TestMarkCategory() {
	ctgID := utils.CreateID()

	c.mockCategories.EXPECT().Mark(gomock.Eq(c.user.ID), gomock.Eq(ctgID), gomock.Eq(models.MarkerRead)).Return(nil)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctgID)

	ctx.SetPath("/v1/categories/:categoryID/mark")

	c.NoError(c.controller.MarkCategory(ctx))
	c.Equal(http.StatusNoContent, rec.Code)
}

func (c *CategoriesControllerSuite) TestMarkCategoryInternalError() {
	c.mockCategories.EXPECT().Mark(gomock.Eq(c.user.ID), gomock.Any(), gomock.Any()).Return(errors.New("error"))

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/mark")

	c.EqualError(
		c.controller.MarkCategory(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *CategoriesControllerSuite) TestMarkUnknownCategory() {
	c.mockCategories.EXPECT().Mark(gomock.Eq(c.user.ID), gomock.Any(), gomock.Any()).Return(services.ErrCategoryNotFound)

	req := httptest.NewRequest(echo.PUT, "/?as=read", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/mark")

	c.EqualError(
		echo.NewHTTPError(http.StatusNotFound),
		c.controller.MarkCategory(ctx).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryEntries() {
	ctgID := utils.CreateID()

	page := models.Page{
		FilterID:       ctgID,
		ContinuationID: "",
		Count:          1,
		Newest:         true,
		Marker:         models.MarkerAny,
	}

	c.mockCategories.EXPECT().Entries(gomock.Eq(c.user.ID), gomock.Eq(page)).Return([]models.Entry{
		{
			ID: utils.CreateID(),
		},
	}, "", nil)

	req := httptest.NewRequest(echo.GET, "/?count=1", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctgID)

	ctx.SetPath("/v1/categories/:categoryID/entries")

	c.NoError(c.controller.GetCategoryEntries(ctx))
}

func (c *CategoriesControllerSuite) TestGetCategoryEntriesBadRequest() {
	req := httptest.NewRequest(echo.GET, "/?count=true", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/entries")

	c.EqualError(
		c.controller.GetCategoryEntries(ctx),
		echo.NewHTTPError(http.StatusBadRequest).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetUnknownCategoryEntries() {
	c.mockCategories.EXPECT().
		Entries(gomock.Eq(c.user.ID), gomock.Any()).
		Return([]models.Entry{}, "", services.ErrCategoryNotFound)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/entries")

	c.EqualError(
		c.controller.GetCategoryEntries(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryEntriesInternalError() {
	c.mockCategories.EXPECT().
		Entries(gomock.Eq(c.user.ID), gomock.Any()).
		Return([]models.Entry{}, "", errors.New("error"))

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/entries")

	c.EqualError(
		c.controller.GetCategoryEntries(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryStats() {
	ctgID := utils.CreateID()

	c.mockCategories.EXPECT().Stats(gomock.Eq(c.user.ID), gomock.Eq(ctgID)).Return(models.Stats{}, nil)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues(ctgID)

	ctx.SetPath("/v1/categories/:categoryID/stats")

	c.NoError(c.controller.GetCategoryStats(ctx))
}

func (c *CategoriesControllerSuite) TestGetUnknownCategoryStats() {
	c.mockCategories.EXPECT().Stats(gomock.Eq(c.user.ID), gomock.Any()).Return(models.Stats{}, services.ErrCategoryNotFound)

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/stats")

	c.EqualError(
		c.controller.GetCategoryStats(ctx),
		echo.NewHTTPError(http.StatusNotFound).Error(),
	)
}

func (c *CategoriesControllerSuite) TestGetCategoryStatsInternalError() {
	c.mockCategories.EXPECT().Stats(gomock.Eq(c.user.ID), gomock.Any()).Return(models.Stats{}, errors.New("error"))

	req := httptest.NewRequest(echo.GET, "/", nil)

	rec := httptest.NewRecorder()
	ctx := c.e.NewContext(req, rec)
	ctx.Set(userContextKey, c.user.ID)
	ctx.SetParamNames("categoryID")
	ctx.SetParamValues("bogus")

	ctx.SetPath("/v1/categories/:categoryID/stats")

	c.EqualError(
		c.controller.GetCategoryStats(ctx),
		echo.NewHTTPError(http.StatusInternalServerError).Error(),
	)
}

func (c *CategoriesControllerSuite) SetupTest() {
	c.ctrl = gomock.NewController(c.T())

	c.e = echo.New()
	c.e.HideBanner = true

	c.user = &models.User{
		ID: utils.CreateID(),
	}

	c.mockCategories = services.NewMockCategories(c.ctrl)

	c.controller = rest.NewCategoriesController(c.mockCategories, c.e)
}

func (c *CategoriesControllerSuite) TearDownTest() {
	c.ctrl.Finish()
}

func TestCategoriesControllerSuite(t *testing.T) {
	suite.Run(t, new(CategoriesControllerSuite))
}
