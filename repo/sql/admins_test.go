package sql

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jmartinezhern/syndication/models"
	"github.com/jmartinezhern/syndication/repo"
	"github.com/jmartinezhern/syndication/utils"
)

type (
	AdminsSuite struct {
		suite.Suite

		repo repo.Admins
		db   *DB
	}
)

func (s *AdminsSuite) TestCreate() {
	adminID := utils.CreateID()

	s.repo.Create(&models.Admin{
		ID:       adminID,
		Username: "gopher",
	})

	admin, found := s.repo.AdminWithID(adminID)
	s.True(found)
	s.Equal(adminID, admin.ID)
	s.Equal("gopher", admin.Username)
}

func (s *AdminsSuite) TestUpdate() {
	adminID := utils.CreateID()
	s.repo.Create(&models.Admin{
		ID:       adminID,
		Username: "gopher",
	})

	err := s.repo.Update(adminID, &models.Admin{Username: "test"})
	s.NoError(err)

	admin, _ := s.repo.AdminWithID(adminID)
	s.Equal("test", admin.Username)
}

func (s *AdminsSuite) TestUpdateMissingAdmin() {
	err := s.repo.Update("", &models.Admin{})
	s.EqualError(err, repo.ErrModelNotFound.Error())
}

func (s *AdminsSuite) TestDelete() {
	adminID := utils.CreateID()

	s.repo.Create(&models.Admin{
		ID:       adminID,
		Username: "gopher",
	})

	err := s.repo.Delete(adminID)
	s.NoError(err)

	_, found := s.repo.AdminWithID(adminID)
	s.False(found)
}

func (s *AdminsSuite) TestDeleteMissingAdmin() {
	err := s.repo.Delete("bogus")
	s.EqualError(err, repo.ErrModelNotFound.Error())
}

func (s *AdminsSuite) TestAdminWithName() {
	admin := models.Admin{
		ID:       utils.CreateID(),
		Username: "gopher",
	}

	s.repo.Create(&admin)

	dbAdmin, found := s.repo.AdminWithName(admin.Username)
	s.True(found)
	s.Equal(admin.Username, dbAdmin.Username)
}

// func (s *AdminsSuite) TestInitialAdmin() {
// 	admin := models.Admin{
// 		ID:    utils.CreateID(),
// 		Username: "gopher",
// 	}
//
// 	s.repo.Create(&admin)
//
// 	_, found := s.repo.InitialUser()
// 	s.True(found)
// }

func (s *AdminsSuite) SetupTest() {
	s.db = NewDB("sqlite3", ":memory:")
	s.repo = NewAdmins(s.db)
}

func (s *AdminsSuite) TearDownTest() {
	s.NoError(s.db.Close())
}

func TestAdminsSuite(t *testing.T) {
	suite.Run(t, new(AdminsSuite))
}
