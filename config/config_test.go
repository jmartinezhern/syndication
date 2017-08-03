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

func (suite *ConfigTestSuite) TestSQLiteConfig() {
	config, err := NewConfig("with_sqlite.toml")
	suite.Require().Nil(err)
	suite.Equal("/tmp/syndication.db", config.Database.Connection)
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
