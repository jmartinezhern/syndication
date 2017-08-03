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

// Package database provides wrapper functions that create and modify objects based on models in an SQL database.
// See the models package for more information on the types of objects database operates on.
package database

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/jinzhu/gorm"
	// GORM dialect packages
	"github.com/dgrijalva/jwt-go"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/scrypt"

	"github.com/chavamee/syndication/models"
)

// Password salt and Hash byte sizes
const (
	PWSaltBytes = 32
	PWHashBytes = 64
)

// DB represents a connectin to a SQL database
type DB struct {
	db         *gorm.DB
	Connection string
	Type       string
}

// NewDB creates a new DB instance
func NewDB(dbType, conn string) (db *DB, err error) {
	gormDB, err := gorm.Open(dbType, conn)
	if err != nil {
		return
	}

	db = &DB{
		Connection: conn,
		Type:       dbType,
	}

	gormDB.AutoMigrate(&models.Feed{})
	gormDB.AutoMigrate(&models.Category{})
	gormDB.AutoMigrate(&models.User{})
	gormDB.AutoMigrate(&models.Entry{})
	gormDB.AutoMigrate(&models.Tag{})
	gormDB.AutoMigrate(&models.APIKey{})

	db.db = gormDB

	return
}

// Close ends connections with the database
func (db *DB) Close() error {
	return db.db.Close()
}

func createPasswordHashAndSalt(password string) (hash []byte, salt []byte, err error) {
	salt = make([]byte, PWSaltBytes)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		return
	}

	hash, err = scrypt.Key([]byte(password), salt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		return
	}

	return
}

// NewUser creates a new User object
func (db *DB) NewUser(username, password string) error {
	user := &models.User{}
	if !db.db.Where("username = ?", username).First(user).RecordNotFound() {
		return Conflict{"User already exists"}
	}

	hash, salt, err := createPasswordHashAndSalt(password)
	if err != nil {
		return err
	}

	// Construct the user system categories
	unctgUUID := uuid.NewV4().String()
	user.Categories = append(user.Categories, models.Category{
		UUID: unctgUUID,
		Name: models.Uncategorized,
	})
	user.UncategorizedCategoryUUID = unctgUUID

	savedUUID := uuid.NewV4().String()
	user.Categories = append(user.Categories, models.Category{
		UUID: savedUUID,
		Name: "Saved",
	})
	user.SavedCategoryUUID = savedUUID

	user.UUID = uuid.NewV4().String()
	user.PasswordHash = hash
	user.PasswordSalt = salt
	user.Username = username

	db.db.Create(&user).Related(&user.Categories)
	return nil
}

// DeleteUser deletes a User object
func (db *DB) DeleteUser(userID string) error {
	user := &models.User{}
	if db.db.Where("uuid = ?", userID).First(user).RecordNotFound() {
		return BadRequest{"User does not exists"}
	}

	db.db.Delete(user)
	return nil
}

// ChangeUserName for user with userID
func (db *DB) ChangeUserName(userID, newName string) error {
	user := &models.User{}
	if db.db.Where("uuid = ?", userID).First(user).RecordNotFound() {
		return BadRequest{"User does not exists"}
	}

	db.db.Model(user).Update("username", newName)
	return nil
}

// ChangeUserPassword for user with userID
func (db *DB) ChangeUserPassword(userID, newPassword string) error {
	user := &models.User{}
	if db.db.Where("uuid = ?", userID).First(user).RecordNotFound() {
		return BadRequest{"User does not exists"}
	}

	hash, salt, err := createPasswordHashAndSalt(newPassword)
	if err != nil {
		return err
	}

	db.db.Model(user).Update(models.User{
		PasswordHash: hash,
		PasswordSalt: salt,
	})
	return nil
}

// Users returns a list of all User entries.
// The parameter fields provides a way to select
// which fields are populated in the returned models.
func (db *DB) Users(fields ...string) (users []models.User) {
	selectFields := "id,uuid"
	if len(fields) != 0 {
		for _, field := range fields {
			selectFields = selectFields + "," + field
		}
	}
	db.db.Select(selectFields).Find(&users)
	return
}

// UserPrimaryKey returns the SQL primary key of a User with a uuid
func (db *DB) UserPrimaryKey(uuid string) (uint, error) {
	user := &models.User{}
	if db.db.First(user, "uuid = ?", uuid).RecordNotFound() {
		return 0, NotFound{"User does not exist"}
	}
	return user.ID, nil
}

// UserWithName returns a User with username
func (db *DB) UserWithName(username string) (user models.User, err error) {
	if db.db.First(&user, "username = ?", username).RecordNotFound() {
		err = NotFound{"User does not exist"}
	}
	return
}

// UserWithUUID returns a User with id
func (db *DB) UserWithUUID(uuid string) (user models.User, err error) {
	if db.db.First(&user, "UUID = ?", uuid).RecordNotFound() {
		err = NotFound{"User does not exist"}
	}
	return
}

// Authenticate a user and return its respective User model if successful
func (db *DB) Authenticate(username, password string) (user models.User, err error) {
	user, err = db.UserWithName(username)
	if err != nil {
		return
	}

	hash, err := scrypt.Key([]byte(password), user.PasswordSalt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		return
	}

	for i, hashByte := range hash {
		if hashByte != user.PasswordHash[i] {
			err = Unauthorized{"Invalid credentials"}
		}
	}

	return
}

// NewAPIKey creates a new APIKey object owned by user
func (db *DB) NewAPIKey(secret string, user *models.User) (models.APIKey, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = user.UUID
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Hour * 72).Unix()

	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return models.APIKey{}, err
	}

	key := &models.APIKey{
		Key:    t,
		User:   *user,
		UserID: user.ID,
	}

	db.db.Model(user).Association("APIKeys").Append(key)

	return *key, nil
}

// KeyBelongsToUser returns true if the given APIKey is owned by user
func (db *DB) KeyBelongsToUser(key *models.APIKey, user *models.User) (bool, error) {
	if key.Key == "" {
		return false, BadRequest{"No key provided"}
	}

	found := !db.db.Model(user).Where("key = ?", key.Key).Related(&models.APIKey{}).RecordNotFound()
	return found, nil
}

// NewFeed creates a new Feed object owned by user
func (db *DB) NewFeed(feed *models.Feed, user *models.User) error {
	feed.UUID = uuid.NewV4().String()

	var err error
	var ctg models.Category
	if feed.Category.UUID != "" {
		ctg, err = db.Category(feed.Category.UUID, user)
		if err != nil {
			return BadRequest{"Feed has invalid category"}
		}
	} else {
		db.db.Model(user).Where("name = ?", models.Uncategorized).Related(&ctg)
	}

	feed.Category = ctg
	feed.CategoryID = ctg.ID
	feed.Category.UUID = ctg.UUID

	db.db.Model(user).Association("Feeds").Append(feed)
	db.db.Model(&ctg).Association("Feeds").Append(feed)

	return nil
}

// Feeds returns a list of all Feeds owned by a user
func (db *DB) Feeds(user *models.User) (feeds []models.Feed) {
	db.db.Model(user).Association("Feeds").Find(&feeds)
	return
}

// FeedsFromCategory returns all Feeds that belong to a category with categoryID
func (db *DB) FeedsFromCategory(categoryID string, user *models.User) (feeds []models.Feed, err error) {
	ctg, err := db.Category(categoryID, user)
	if err != nil {
		return
	}

	db.db.Model(ctg).Association("Feeds").Find(&feeds)
	return
}

// Feed returns a Feed with id and owned by user
func (db *DB) Feed(id string, user *models.User) (feed models.Feed, err error) {
	if db.db.Model(user).Where("uuid = ?", id).Related(&feed).RecordNotFound() {
		err = NotFound{"Feed does not exist"}
		return
	}

	db.db.Model(&feed).Related(&feed.Category)
	return
}

// DeleteFeed with id and owned by user
func (db *DB) DeleteFeed(id string, user *models.User) error {
	foundFeed := &models.Feed{}
	if !db.db.Model(user).Where("uuid = ?", id).Related(foundFeed).RecordNotFound() {
		db.db.Delete(foundFeed)
		return nil
	}
	return NotFound{"Feed does not exist"}
}

// EditFeed owned by user
func (db *DB) EditFeed(feed *models.Feed, user *models.User) error {
	foundFeed := &models.Feed{}
	if !db.db.Model(user).Related(foundFeed, "uuid = ?", feed.UUID).RecordNotFound() {
		foundFeed.Title = feed.Title
		db.db.Model(feed).Save(foundFeed)
		return nil
	}
	return NotFound{"Feed does not exist"}
}

// NewCategory creates a new Category object owned by user
func (db *DB) NewCategory(ctg *models.Category, user *models.User) error {
	if ctg.Name == "" {
		return BadRequest{"Category name should not be empty"}
	}

	tmpCtg := &models.Category{}
	if db.db.Model(user).Where("name = ?", ctg.Name).Related(tmpCtg).RecordNotFound() {
		ctg.UUID = uuid.NewV4().String()
		db.db.Model(user).Association("Categories").Append(ctg)
		return nil
	}

	return Conflict{"Category already exists"}
}

// EditCategory owned by user
func (db *DB) EditCategory(ctg *models.Category, user *models.User) error {
	foundCtg := &models.Category{}
	if !db.db.Model(user).Where("uuid = ?", ctg.UUID).Related(foundCtg).RecordNotFound() {
		foundCtg.Name = ctg.Name
		db.db.Model(ctg).Save(foundCtg)
		return nil
	}
	return NotFound{"Category does not exist"}
}

// DeleteCategory with id and owned by user
func (db *DB) DeleteCategory(id string, user *models.User) error {
	if id == user.UncategorizedCategoryUUID || id == user.SavedCategoryUUID {
		return BadRequest{"Cannot delete system categories"}
	}

	ctg := &models.Category{}
	if db.db.Model(user).Where("uuid = ?", id).Related(ctg).RecordNotFound() {
		return NotFound{"Category does not exist"}
	}

	db.db.Delete(ctg)
	return nil
}

// Category returns a Category with id and owned by user
func (db *DB) Category(id string, user *models.User) (ctg models.Category, err error) {
	if db.db.Model(user).Where("uuid = ?", id).Related(&ctg).RecordNotFound() {
		err = NotFound{"Category does not exist"}
	}
	return
}

// Categories returns a list of all Categories owned by user
func (db *DB) Categories(user *models.User) (categories []models.Category) {
	db.db.Model(user).Association("Categories").Find(&categories)
	return
}

// ChangeFeedCategory changes the category a feed belongs to
func (db *DB) ChangeFeedCategory(feedID string, ctgID string, user *models.User) error {
	feed := &models.Feed{}
	if db.db.Model(user).Where("uuid = ?", feedID).Related(feed).RecordNotFound() {
		return NotFound{"Feed does not exist"}
	}

	prevCtg := &models.Category{
		ID: feed.CategoryID,
	}

	db.db.First(prevCtg)

	db.db.Model(prevCtg).Association("Feeds").Delete(feed)

	newCtg := &models.Category{}
	if db.db.Model(user).Where("uuid = ?", ctgID).Related(newCtg).RecordNotFound() {
		return NotFound{"Category does not exist"}
	}

	db.db.Model(newCtg).Association("Feeds").Append(feed)

	return nil
}

// NewEntry creates a new Entry object owned by user
func (db *DB) NewEntry(entry *models.Entry, user *models.User) error {
	if entry.Feed.UUID == "" {
		return BadRequest{"Entry should have a feed"}
	}

	feed := models.Feed{}
	if db.db.Model(user).Where("uuid = ?", entry.Feed.UUID).Related(&feed).RecordNotFound() {
		return NotFound{"Feed does not exist"}
	}

	entry.UUID = uuid.NewV4().String()
	entry.Feed = feed
	entry.FeedID = feed.ID

	db.db.Model(user).Association("Entries").Append(entry)
	db.db.Model(&feed).Association("Entries").Append(entry)

	return nil
}

// NewEntries creates multiple new Entry objects which
// are all owned by feed with feedUUID and user
func (db *DB) NewEntries(entries []models.Entry, feed models.Feed, user *models.User) error {
	if feed.UUID == "" {
		return BadRequest{"Entry should have a feed"}
	}

	if len(entries) == 0 {
		return nil
	}

	if db.db.Model(user).Where("uuid = ?", feed.UUID).Related(&feed).RecordNotFound() {
		return NotFound{"Feed does not exist"}
	}

	for _, entry := range entries {
		entry.UUID = uuid.NewV4().String()
		entry.Feed = feed
		entry.FeedID = feed.ID

		db.db.Model(user).Association("Entries").Append(&entry)
		db.db.Model(&feed).Association("Entries").Append(&entry)
	}

	return nil
}

// Entry returns an Entry with id and owned by user
func (db *DB) Entry(id string, user *models.User) (entry models.Entry, err error) {
	if db.db.Model(user).Where("uuid = ?", id).Related(&entry).RecordNotFound() {
		err = NotFound{"Feed does not exists"}
		return
	}

	db.db.Model(&entry).Related(&entry.Feed)
	return
}

// EntryWithGUIDExists returns true if an Entry exists with the given guid and is owned by user
func (db *DB) EntryWithGUIDExists(guid string, user *models.User) bool {
	return !db.db.Model(user).Where("guid = ?", guid).Related(&models.Entry{}).RecordNotFound()
}

// Entries returns a list of all entries owned by user
func (db *DB) Entries(orderByDesc bool, marker models.Marker, user *models.User) (entries []models.Entry, err error) {
	if marker == models.None {
		err = BadRequest{"Request should include a valid marker"}
		return
	}

	query := db.db.Model(user)
	if marker != models.Any {
		query = query.Where("mark = ?", marker)
	}

	query.Association("Entries").Find(&entries)
	return
}

// EntriesFromFeed returns all Entries that belong to a feed with feedID
func (db *DB) EntriesFromFeed(feedID string, orderByDesc bool, marker models.Marker, user *models.User) (entries []models.Entry, err error) {
	if marker == models.None {
		err = BadRequest{"Request should include a valid marker"}
		return
	}

	feed := &models.Feed{}
	if db.db.Model(user).Where("uuid = ?", feedID).Related(feed).RecordNotFound() {
		err = NotFound{"Feed not found"}
		return
	}

	query := db.db.Model(feed)
	if marker != models.Any {
		query = query.Where("mark = ?", marker)
	}

	query.Association("Entries").Find(&entries)

	return
}

// EntriesFromCategory returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func (db *DB) EntriesFromCategory(categoryID string, orderByDesc bool, marker models.Marker, user *models.User) (entries []models.Entry, err error) {
	if marker == models.None {
		err = BadRequest{"Request should include a valid marker"}
		return
	}

	category := &models.Category{}
	if db.db.Model(user).Where("uuid = ?", categoryID).Related(category).RecordNotFound() {
		err = NotFound{"Category not found"}
		return
	}

	var feeds []models.Feed
	db.db.Model(category).Related(&feeds)

	var order *gorm.DB
	if orderByDesc {
		order = db.db.Model(user).Order("created_at DESC")
	} else {
		order = db.db.Model(user).Order("created_at ASC")
	}

	if marker != models.Any {
		order = order.Where("mark = ?", marker)
	}

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	order.Where("feed_id in (?)", feedIds).Association("Entries").Find(&entries)
	return
}

// EntriesFromTag returns all Entries which are tagged with tagID
func (db *DB) EntriesFromTag(tagID string, makrer models.Marker, orderByDesc bool, user *models.User) (entries []models.Entry, err error) {
	// TODO
	return
}

// CategoryStats returns all Stats for a Category with the given id and that is owned by user
func (db *DB) CategoryStats(id string, user *models.User) (stats models.Stats, err error) {
	ctg := &models.Category{}
	if db.db.Model(user).Where("uuid = ?", id).Related(ctg).RecordNotFound() {
		err = NotFound{"Category not found"}
		return
	}

	var feeds []models.Feed
	db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	query := db.db.Model(user).Where("feed_id in (?)", feedIds)

	stats.Unread = query.Where("mark = ?", models.Unread).Association("Entries").Count()
	stats.Read = query.Where("mark = ?", models.Read).Association("Entries").Count()
	stats.Saved = query.Where("saved = ?", true).Association("Entries").Count()
	stats.Total = query.Association("Entries").Count()
	return
}

// FeedStats returns all Stats for a Feed with the given id and that is owned by user
func (db *DB) FeedStats(id string, user *models.User) (stats models.Stats, err error) {
	feed := &models.Feed{}
	if db.db.Model(user).Where("uuid = ?", id).Related(feed).RecordNotFound() {
		err = NotFound{"Feed not found"}
		return
	}

	stats.Unread = db.db.Model(user).Where("feed_id = ? AND mark = ?", feed.ID, models.Unread).Association("Entries").Count()
	stats.Read = db.db.Model(user).Where("feed_id = ? AND mark = ?", feed.ID, models.Read).Association("Entries").Count()
	stats.Saved = db.db.Model(user).Where("feed_id = ? AND saved = ?", feed.ID, true).Association("Entries").Count()
	stats.Total = db.db.Model(user).Where("feed_id = ?", feed.ID).Association("Entries").Count()
	return
}

// Stats returns all Stats for the given user
func (db *DB) Stats(user *models.User) (stats models.Stats) {
	stats.Unread = db.db.Model(user).Where("mark = ?", models.Unread).Association("Entries").Count()
	stats.Read = db.db.Model(user).Where("mark = ?", models.Read).Association("Entries").Count()
	stats.Saved = db.db.Model(user).Where("saved = ?", true).Association("Entries").Count()
	stats.Total = db.db.Model(user).Association("Entries").Count()
	return
}

// MarkFeed applies marker to a Feed with id and owned by user
func (db *DB) MarkFeed(id string, marker models.Marker, user *models.User) error {
	feed, err := db.Feed(id, user)
	if err != nil {
		return err
	}

	db.db.Model(&models.Entry{}).Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).Update(models.Entry{Mark: marker})
	return nil
}

// MarkCategory applies marker to a category with id and owned by user
func (db *DB) MarkCategory(id string, marker models.Marker, user *models.User) error {
	ctg, err := db.Category(id, user)
	if err != nil {
		return err
	}

	var feeds []models.Feed
	db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	db.db.Model(&models.Entry{}).Where("user_id = ?", user.ID).Where("feed_id in (?)", feedIds).Update(models.Entry{Mark: marker})
	return nil
}

// MarkEntry applies marker to an entry with id and owned by user
func (db *DB) MarkEntry(id string, marker models.Marker, user *models.User) error {
	entry, err := db.Entry(id, user)
	if err != nil {
		return err
	}

	db.db.Model(&entry).Update(models.Entry{Mark: marker})
	return nil
}

// DeleteAll records in the database
func (db *DB) DeleteAll() {
	db.db.Delete(&models.Feed{})
	db.db.Delete(&models.Category{})
	db.db.Delete(&models.User{})
	db.db.Delete(&models.Entry{})
	db.db.Delete(&models.Tag{})
	db.db.Delete(&models.APIKey{})
}
