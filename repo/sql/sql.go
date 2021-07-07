/*
 *   Copyright (C) 2021. Jorge Martinez Hernandez
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU Affero General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU Affero General Public License for more details.
 *
 *   You should have received a copy of the GNU Affero General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package sql

import (
	"github.com/jinzhu/gorm"

	// GORM dialect packages
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/jmartinezhern/syndication/models"
)

func AutoMigrateTables(db *gorm.DB) {
	db.AutoMigrate(&models.Feed{})
	db.AutoMigrate(&models.Category{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Entry{})
	db.AutoMigrate(&models.Tag{})
	db.AutoMigrate(&models.APIKey{})
}
