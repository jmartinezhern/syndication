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

// Package database provides routines to operate on Syndications SQL database
// using models defined in the models package to map data in said database.
package database

import (
	"encoding/base64"
	"errors"
	mathRand "math/rand"
	"strconv"
	"time"

	"github.com/jmartinezhern/syndication/models"

	"github.com/jinzhu/gorm"
	// GORM dialect packages
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Password salt and Hash byte sizes
const (
	PWSaltBytes = 32
	PWHashBytes = 64
)

type (
	// DB represents a connection to a SQL database
	DB struct {
		db *gorm.DB
	}
)

var (
	// ErrModelNotFound signals that an operation was attempted
	// on a model that is not found in the database.
	ErrModelNotFound = errors.New("Model not found in database")
)

var (
	lastTimeIDWasCreated int64
	random32Int          uint32
	defaultInstance      *DB
)

// Init initializes a database instance
func Init(dbType, connection string) error {
	var err error
	defaultInstance, err = NewDB(dbType, connection)
	return err
}

// NewDB creates a new DB instance
func NewDB(dbType, connection string) (*DB, error) {
	gormDB, err := gorm.Open(dbType, connection)
	if err != nil {
		return nil, err
	}

	gormDB.AutoMigrate(&models.Feed{})
	gormDB.AutoMigrate(&models.Category{})
	gormDB.AutoMigrate(&models.User{})
	gormDB.AutoMigrate(&models.Entry{})
	gormDB.AutoMigrate(&models.Tag{})
	gormDB.AutoMigrate(&models.APIKey{})

	db := &DB{
		db: gormDB,
	}

	return db, nil
}

// Close ends connections with the database
func (db *DB) Close() error {
	return db.db.Close()
}

// Close ends connections with the database
func Close() error {
	return defaultInstance.Close()
}

// Stats returns all Stats for the given user
func (db *DB) Stats(user models.User) models.Stats {
	stats := models.Stats{}

	stats.Unread = db.db.Model(&user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = db.db.Model(&user).Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = db.db.Model(&user).Where("saved = ?", true).Association("Entries").Count()
	stats.Total = db.db.Model(&user).Association("Entries").Count()

	return stats
}

// Stats returns all Stats for the given user
func Stats(user models.User) models.Stats {
	return defaultInstance.Stats(user)
}

func createAPIID() string {
	currentTime := time.Now().Unix()
	duplicateTime := (lastTimeIDWasCreated == currentTime)
	lastTimeIDWasCreated = currentTime

	if !duplicateTime {
		random32Int = mathRand.Uint32() % 16
	} else {
		random32Int++
	}

	idStr := strconv.FormatInt(currentTime+int64(random32Int), 10)
	return base64.StdEncoding.EncodeToString([]byte(idStr))
}
