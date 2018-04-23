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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	mathRand "math/rand"
	"strconv"
	"time"

	"github.com/varddum/syndication/models"

	"github.com/jinzhu/gorm"
	// GORM dialect packages
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"golang.org/x/crypto/scrypt"

	log "github.com/sirupsen/logrus"
)

// Password salt and Hash byte sizes
const (
	PWSaltBytes = 32
	PWHashBytes = 64
)

type (
	// DB represents a connectin to a SQL database
	DB struct {
		db *gorm.DB
	}
)

var (
	// ErrModelNotFound signals that an operation was attempted
	// on a model that is not found in the database.
	ErrModelNotFound = errors.New("Model not found in database")
)

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

var lastTimeIDWasCreated int64
var random32Int uint32

// Close ends connections with the database
func (db *DB) Close() error {
	return db.db.Close()
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

func createPasswordHashAndSalt(password string) ([]byte, []byte) {
	var err error

	salt := make([]byte, PWSaltBytes)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err) // We must be able to read from random
	}

	hash, err := scrypt.Key([]byte(password), salt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		panic(err) // We must never get an error
	}

	return hash, salt
}

// NewUser creates a new User object
func (db *DB) NewUser(username, password string) models.User {
	user := models.User{}
	hash, salt := createPasswordHashAndSalt(password)

	// Construct the user system categories
	unctgAPIID := createAPIID()
	user.Categories = append(user.Categories, models.Category{
		APIID: unctgAPIID,
		Name:  models.Uncategorized,
	})
	user.UncategorizedCategoryAPIID = unctgAPIID

	user.APIID = createAPIID()
	user.PasswordHash = hash
	user.PasswordSalt = salt
	user.Username = username

	db.db.Create(&user).Related(&user.Categories)

	return user
}

// DeleteUser with apiID
func (db *DB) DeleteUser(apiID string) error {
	user, found := db.UserWithAPIID(apiID)
	if !found {
		return ErrModelNotFound
	}

	db.db.Delete(&user)
	return nil
}

// ChangeUserName for user with userID
func (db *DB) ChangeUserName(apiID, newName string) error {
	user, found := db.UserWithAPIID(apiID)
	if !found {
		return ErrModelNotFound
	}

	db.db.Model(&user).Update("username", newName)
	return nil
}

// ChangeUserPassword for user with apiID
func (db *DB) ChangeUserPassword(apiID, newPassword string) error {
	user, found := db.UserWithAPIID(apiID)
	if !found {
		return ErrModelNotFound
	}

	hash, salt := createPasswordHashAndSalt(newPassword)

	db.db.Model(&user).Update(models.User{
		PasswordHash: hash,
		PasswordSalt: salt,
	})

	return nil
}

// Users returns a list of all User entries.
// The parameter fields provides a way to select
// which fields are populated in the returned models.
func (db *DB) Users(fields ...string) []models.User {
	users := []models.User{}

	selectFields := "id,api_id"
	if len(fields) != 0 {
		for _, field := range fields {
			selectFields = selectFields + "," + field
		}
	}
	db.db.Select(selectFields).Find(&users)

	return users
}

// UserWithName returns a User with username
func (db *DB) UserWithName(username string) (user models.User, found bool) {
	found = !db.db.First(&user, "username = ?", username).RecordNotFound()
	return
}

// UserWithCredentials returns a user whose credentials matches the ones given.
// Ok will be false if the user was not found or if the credentials did not match.
// This function assumes that the password does not exceed scrypt's payload size.
func (db *DB) UserWithCredentials(username, password string) (models.User, bool) {
	foundUser, ok := db.UserWithName(username)
	if !ok {
		return models.User{}, ok
	}

	hash, err := scrypt.Key([]byte(password), foundUser.PasswordSalt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		log.Error("Failed to generate a hash: ", err)
		return models.User{}, false
	}

	for i, hashByte := range hash {
		if hashByte != foundUser.PasswordHash[i] {
			return models.User{}, false
		}
	}

	return foundUser, true
}

// UserWithAPIID returns a User with id
func (db *DB) UserWithAPIID(apiID string) (user models.User, found bool) {
	found = !db.db.First(&user, "api_id = ?", apiID).RecordNotFound()
	return
}
