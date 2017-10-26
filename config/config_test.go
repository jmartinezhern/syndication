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

package config

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type (
	ConfigTestSuite struct {
		suite.Suite
	}
)

func (suite *ConfigTestSuite) SetupTest() {
}

func (suite *ConfigTestSuite) TearDown() {
}

func (suite *ConfigTestSuite) TestNewConfig() {
	config, err := NewConfig("simple.toml")
	suite.Nil(err)
	suite.Equal("/tmp/syndication.db", config.Database.Connection)
}

func (suite *ConfigTestSuite) TestNewConfigOnInvalidPath() {
	_, err := NewConfig("bogus.toml")
	suite.NotNil(err)
}

func (suite *ConfigTestSuite) TestNewConfigWithSecretFile() {
	_, err := NewConfig("simple_with_file_secret.toml")
	suite.Nil(err)
}

func (suite *ConfigTestSuite) TestNewConfigWithBadSecretFile() {
	config, err := NewConfig("simple_with_file_secret.toml")
	suite.Require().Nil(err)

	config.Server.AuthSecreteFilePath = "bogus"

	suite.NotNil(config.verifyConfig())
}

func (suite *ConfigTestSuite) TestNewInvalidConfig() {
	_, err := NewConfig("invalid.toml")
	suite.IsType(err, InvalidFieldValue{})
}

func (suite *ConfigTestSuite) TestSQLiteConfig() {
	config, err := NewConfig("sqlite.toml")
	suite.Require().Nil(err)
	suite.Equal("/tmp/syndication.db", config.Database.Connection)
}

func (suite *ConfigTestSuite) TestBadSQLiteConfig() {
	config, err := NewConfig("sqlite.toml")
	suite.Require().Nil(err)

	config.Database = Database{}

	config.Databases["sqlite"] = Database{"sqlite", true, ""}
	suite.NotNil(config.verifyConfig())

	config.Database = Database{}

	config.Databases["sqlite"] = Database{"sqlite", true, "bogus"}
	suite.NotNil(config.verifyConfig())
}

func (suite *ConfigTestSuite) TestMySQLConfig() {
	_, err := NewConfig("mysql.toml")
	suite.Require().Nil(err)
}

func (suite *ConfigTestSuite) TestBadMySQLConfig() {
	_, err := NewConfig("bad_mysql.toml")
	suite.Require().NotNil(err)
}

func (suite *ConfigTestSuite) TestPostgresConfig() {
	_, err := NewConfig("postgres.toml")
	suite.Require().Nil(err)
}

func (suite *ConfigTestSuite) TestUnknownDB() {
	_, err := NewConfig("with_unknown_db.toml")
	suite.Require().Nil(err)
}

func (suite *ConfigTestSuite) TestNoDB() {
	_, err := NewConfig("no_db.toml")
	suite.Require().NotNil(err)
}

func (suite *ConfigTestSuite) TestNoDBEnabled() {
	_, err := NewConfig("no_db_enabled.toml")
	suite.Require().NotNil(err)
}

func (suite *ConfigTestSuite) TestErrors() {
	invFieldErr := InvalidFieldValue{"Invalid Field"}
	suite.Equal("Invalid Field", invFieldErr.Error())

	fileSysErr := FileSystemError{"FileSystem Error"}
	suite.Equal("FileSystem Error", fileSysErr.Error())

	parseErr := ParsingError{"Parsing Error"}
	suite.Equal("Parsing Error", parseErr.Error())
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
