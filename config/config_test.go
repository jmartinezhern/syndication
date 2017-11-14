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
	"os"
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
	dir, err := os.Getwd()
	suite.Require().Nil(err)

	err = os.Symlink(dir+"/sample_secret", "/tmp/sample_secret")
	suite.Require().Nil(err)
	defer os.Remove("/tmp/sample_secret")

	_, err = NewConfig("simple_with_file_secret.toml")
	suite.Nil(err)
}

func (suite *ConfigTestSuite) TestNewConfigWithBadSecretFile() {
	dir, err := os.Getwd()
	suite.Require().Nil(err)

	err = os.Symlink(dir+"/sample_secret", "/tmp/sample_secret")
	suite.Require().Nil(err)
	defer os.Remove("/tmp/sample_secret")

	config, err := NewConfig("simple_with_file_secret.toml")
	suite.Require().Nil(err)

	config.Server.AuthSecreteFilePath = "bogus"

	suite.NotNil(config.verifyConfig())

}

func (suite *ConfigTestSuite) TestMinimalConfig() {
	config, err := NewConfig("simple.toml")
	suite.Require().Nil(err)

	suite.Equal(DefaultSyncConfig, config.Sync)
	suite.Equal(DefaultAdminConfig, config.Admin)
}

func (suite *ConfigTestSuite) TestValidPlugins() {
	_, err := os.Stat("/tmp/libtest.so")
	if err != nil {
		os.OpenFile("/tmp/libtest.so", os.O_RDONLY|os.O_CREATE, 0666)
	}

	config, err := NewConfig("valid_plugins.toml")
	suite.Require().Nil(err)
	suite.Len(config.Plugins, 1)
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

	config.Databases["sqlite"] = Database{"sqlite", true, "", Duration{0}}
	suite.NotNil(config.verifyConfig())

	config.Database = Database{}

	config.Databases["sqlite"] = Database{"sqlite", true, "bogus", Duration{0}}
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

func (suite *ConfigTestSuite) TestInvalidPlugins() {
	config, err := NewConfig("invalid_plugins.toml")
	suite.Require().Nil(err)
	suite.Empty(config.Plugins)
}

func (suite *ConfigTestSuite) TestNoDB() {
	_, err := NewConfig("no_db.toml")
	suite.Require().NotNil(err)
}

func (suite *ConfigTestSuite) TestNoDBEnabled() {
	_, err := NewConfig("no_db_enabled.toml")
	suite.Require().NotNil(err)
}

func (suite *ConfigTestSuite) TestSyncConfig() {
	_, err := NewConfig("simple_sync.toml")
	suite.Require().Nil(err)
}

func (suite *ConfigTestSuite) TestShortSyncInterval() {
	_, err := NewConfig("invalid_sync.toml")
	suite.Require().NotNil(err)
}

func (suite *ConfigTestSuite) TestInvalidAdmin() {
	_, err := NewConfig("invalid_admin.toml")
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
