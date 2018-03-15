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

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	// GORM dialect packages
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"golang.org/x/crypto/scrypt"

	log "github.com/sirupsen/logrus"
	"github.com/varddum/syndication/config"
)

// Password salt and Hash byte sizes
const (
	PWSaltBytes = 32
	PWHashBytes = 64
)

type (
	// DB represents a connectin to a SQL database
	DB struct {
		db     *gorm.DB
		config config.Database
	}
)

var (
	// ErrModelNotFound signals that an operation was attempted
	// on a model that is not found in the database.
	ErrModelNotFound = errors.New("Model not found in database")
)

// NewDB creates a new DB instance
func NewDB(conf config.Database) (db *DB, err error) {
	gormDB, err := gorm.Open(conf.Type, conf.Connection)
	if err != nil {
		return
	}

	db = &DB{
		config: conf,
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

var lastTimeIDWasCreated int64
var random32Int uint32

// Close ends connections with the database
func (db *DB) Close() error {
	return db.db.Close()
}

func (db *DB) createFeed(feed *models.Feed, ctg *models.Category, user *models.User) {
	feed.APIID = createAPIID()

	if ctg != nil {
		feed.Category = *ctg
		feed.CategoryID = ctg.ID
		feed.Category.APIID = ctg.APIID

		db.db.Model(user).Association("Feeds").Append(feed)
		db.db.Model(ctg).Association("Feeds").Append(feed)
	} else {
		db.db.Model(user).Association("Feeds").Append(feed)
	}
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

func createPasswordHashAndSalt(password string) (hash []byte, salt []byte) {
	var err error

	salt = make([]byte, PWSaltBytes)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err) // We must be able to read from random
	}

	hash, err = scrypt.Key([]byte(password), salt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		panic(err) // We must never get an error
	}

	return
}

// NewUser creates a new User object
func (db *DB) NewUser(username, password string) (user models.User) {
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
	return
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
func (db *DB) Users(fields ...string) (users []models.User) {
	selectFields := "id,api_id"
	if len(fields) != 0 {
		for _, field := range fields {
			selectFields = selectFields + "," + field
		}
	}
	db.db.Select(selectFields).Find(&users)
	return
}

// UserWithName returns a User with username
func (db *DB) UserWithName(username string) (user models.User, found bool) {
	found = !db.db.First(&user, "username = ?", username).RecordNotFound()
	return
}

// CategoryWithName returns a Category that has a matching name and belongs to the given user
func (db *DB) CategoryWithName(name string, user *models.User) (ctg models.Category, found bool) {
	found = !db.db.Model(user).Where("name = ?", name).Related(&ctg).RecordNotFound()
	return
}

// TagWithName returns a Tag that has a matching name and belongs to the given user
func (db *DB) TagWithName(name string, user *models.User) (tag models.Tag, found bool) {
	found = !db.db.Model(user).Where("name = ?", name).Related(&tag).RecordNotFound()
	return
}

// UserWithAPIID returns a User with id
func (db *DB) UserWithAPIID(apiID string) (user models.User, found bool) {
	found = !db.db.First(&user, "api_id = ?", apiID).RecordNotFound()
	return
}

// EntryWithAPIID returns an Entry with id that belongs to user
func (db *DB) EntryWithAPIID(apiID string, user *models.User) (entry models.Entry, found bool) {
	found = !db.db.Model(user).Where("api_id = ?", apiID).Related(&entry).RecordNotFound()
	return
}

// TagWithAPIID returns a Tag with id that belongs to user
func (db *DB) TagWithAPIID(apiID string, user *models.User) (tag models.Tag, found bool) {
	found = !db.db.Model(user).Where("api_id = ?", apiID).Related(&tag).RecordNotFound()
	return
}

// UserWithCredentials returns a user whose credentials matches the ones given.
// Ok will be false if the user was not found or if the credentials did not match.
// This function asssumes that the password does not exceed scrypt's payload size.
func (db *DB) UserWithCredentials(username, password string) (user models.User, ok bool) {
	foundUser, ok := db.UserWithName(username)
	if !ok {
		return
	}

	hash, err := scrypt.Key([]byte(password), foundUser.PasswordSalt, 1<<14, 8, 1, PWHashBytes)
	if err != nil {
		log.Error("Failed to generate a hash: ", err)
		ok = false
		return
	}

	for i, hashByte := range hash {
		if hashByte != foundUser.PasswordHash[i] {
			ok = false
			return
		}
	}

	user = foundUser
	ok = true
	return
}

// NewAPIKey creates a new APIKey object owned by user
func (db *DB) NewAPIKey(secret string, user *models.User) (models.APIKey, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = user.APIID
	claims["admin"] = false
	claims["exp"] = time.Now().Add(db.config.APIKeyExpiration.Duration).Unix()

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
func (db *DB) KeyBelongsToUser(key models.APIKey, user *models.User) bool {
	return !db.db.Model(user).Where("key = ?", key.Key).Related(&key).RecordNotFound()
}

// NewFeedWithCategory creates a new feed associated to a category with the given API ID
func (db *DB) NewFeedWithCategory(title, subscription, ctgID string, user *models.User) (feed models.Feed, err error) {
	ctg, found := db.CategoryWithAPIID(ctgID, user)
	if !found {
		err = ErrModelNotFound
		return
	}

	feed.Title = title
	feed.Subscription = subscription

	db.createFeed(&feed, &ctg, user)

	return
}

// NewFeed creates a new Feed object owned by user
func (db *DB) NewFeed(title, subscription string, user *models.User) (feed models.Feed) {
	feed.Title = title
	feed.Subscription = subscription

	ctg := models.Category{}
	db.db.Model(user).Where("name = ?", models.Uncategorized).Related(&ctg)

	db.createFeed(&feed, &ctg, user)

	return
}

// Feeds returns a list of all Feeds owned by a user
func (db *DB) Feeds(user *models.User) (feeds []models.Feed) {
	db.db.Model(user).Association("Feeds").Find(&feeds)
	return
}

// FeedsFromCategory returns all Feeds that belong to a category with categoryID
func (db *DB) FeedsFromCategory(categoryID string, user *models.User) (feeds []models.Feed) {
	if ctg, found := db.CategoryWithAPIID(categoryID, user); found {
		db.db.Model(ctg).Association("Feeds").Find(&feeds)
	}
	return
}

// FeedWithAPIID returns a Feed with id and owned by user
func (db *DB) FeedWithAPIID(id string, user *models.User) (feed models.Feed, found bool) {
	found = !db.db.Model(user).Where("api_id = ?", id).Related(&feed).RecordNotFound()
	if found {
		db.db.Model(&feed).Related(&feed.Category)
	}
	return
}

// DeleteFeed with id and owned by user
func (db *DB) DeleteFeed(id string, user *models.User) error {
	foundFeed := &models.Feed{}
	if !db.db.Model(user).Where("api_id = ?", id).Related(foundFeed).RecordNotFound() {
		db.db.Delete(foundFeed)
		return nil
	}
	return ErrModelNotFound
}

// EditFeed owned by user
func (db *DB) EditFeed(feed *models.Feed, user *models.User) error {
	if dbFeed, found := db.FeedWithAPIID(feed.APIID, user); found {
		db.db.Model(&dbFeed).Updates(feed)
		return nil
	}
	return ErrModelNotFound
}

// NewCategory creates a new Category object owned by user
func (db *DB) NewCategory(name string, user *models.User) models.Category {
	ctg := models.Category{
		Name:  name,
		APIID: createAPIID(),
	}
	db.db.Model(user).Association("Categories").Append(&ctg)
	return ctg
}

// EditCategory owned by user
func (db *DB) EditCategory(ctg *models.Category, user *models.User) error {
	foundCtg := &models.Category{}
	if !db.db.Model(user).Where("api_id = ?", ctg.APIID).Related(foundCtg).RecordNotFound() {
		foundCtg.Name = ctg.Name
		db.db.Model(ctg).Save(foundCtg)
		return nil
	}
	return ErrModelNotFound
}

// DeleteCategory with id and owned by user
func (db *DB) DeleteCategory(id string, user *models.User) error {
	ctg := &models.Category{}
	if db.db.Model(user).Where("api_id = ?", id).Related(ctg).RecordNotFound() {
		return ErrModelNotFound
	}

	db.db.Delete(ctg)
	return nil
}

// CategoryWithAPIID returns a Category with id and owned by user
func (db *DB) CategoryWithAPIID(id string, user *models.User) (ctg models.Category, found bool) {
	found = !db.db.Model(user).Where("api_id = ?", id).Related(&ctg).RecordNotFound()
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
	if db.db.Model(user).Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return ErrModelNotFound
	}

	prevCtg := &models.Category{
		ID: feed.CategoryID,
	}

	db.db.First(prevCtg)

	db.db.Model(prevCtg).Association("Feeds").Delete(feed)

	newCtg := &models.Category{}
	if db.db.Model(user).Where("api_id = ?", ctgID).Related(newCtg).RecordNotFound() {
		return ErrModelNotFound
	}

	db.db.Model(newCtg).Association("Feeds").Append(feed)

	return nil
}

// NewEntry creates a new Entry object owned by user
func (db *DB) NewEntry(entry models.Entry, feedID string, user *models.User) (models.Entry, error) {
	feed, found := db.FeedWithAPIID(feedID, user)
	if !found {
		return models.Entry{}, ErrModelNotFound
	}

	entry.APIID = createAPIID()
	entry.Feed = feed
	entry.FeedID = feed.ID

	db.db.Model(user).Association("Entries").Append(entry)
	db.db.Model(&feed).Association("Entries").Append(entry)

	return entry, nil
}

// NewEntries creates multiple new Entry objects which
// are all owned by feed with feedAPIID and user
func (db *DB) NewEntries(entries []models.Entry, feedID string, user *models.User) ([]models.Entry, error) {
	if len(entries) == 0 {
		// Nothing to do
		return nil, nil
	}

	feed, found := db.FeedWithAPIID(feedID, user)
	if !found {
		return nil, ErrModelNotFound
	}

	for i, entry := range entries {
		entry.APIID = createAPIID()

		db.db.Model(user).Association("Entries").Append(&entry)
		db.db.Model(&feed).Association("Entries").Append(&entry)

		entries[i] = entry
	}

	return entries, nil
}

// EntryWithGUIDExists returns true if an Entry exists with the given guid and is owned by user
func (db *DB) EntryWithGUIDExists(guid string, feedID string, user *models.User) bool {
	userModel := db.db.Model(user)
	feed := new(models.Feed)
	if userModel.Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return false
	}

	return !userModel.Where("guid = ? AND feed_id = ?", guid, feed.ID).Related(&models.Entry{}).RecordNotFound()
}

// Entries returns a list of all entries owned by user
func (db *DB) Entries(orderByNewest bool, marker models.Marker, user *models.User) (entries []models.Entry) {
	if marker == models.None {
		return
	}

	query := db.db.Model(user)
	if marker != models.Any {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Association("Entries").Find(&entries)
	return
}

// EntriesFromFeed returns all Entries that belong to a feed with feedID
func (db *DB) EntriesFromFeed(feedID string, orderByNewest bool, marker models.Marker, user *models.User) (entries []models.Entry) {
	if marker == models.None {
		return
	}

	feed := &models.Feed{}
	if db.db.Model(user).Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return
	}

	query := db.db.Model(user)
	if marker != models.Any {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Where("feed_id = ?", feed.ID).Association("Entries").Find(&entries)

	return
}

// EntriesFromCategory returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func (db *DB) EntriesFromCategory(categoryID string, orderByNewest bool, marker models.Marker, user *models.User) (entries []models.Entry) {
	if marker == models.None {
		return
	}

	category := &models.Category{}
	if db.db.Model(user).Where("api_id = ?", categoryID).Related(category).RecordNotFound() {
		return
	}

	var feeds []models.Feed
	db.db.Model(category).Related(&feeds)

	query := db.db.Model(user)
	if marker != models.Any {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	query.Where("feed_id in (?)", feedIds).Association("Entries").Find(&entries)
	return
}

// NewTag creates a new Tag object owned by user
func (db *DB) NewTag(name string, user *models.User) models.Tag {
	tag := models.Tag{}
	if db.db.Model(user).Where("name = ?", name).Related(&tag).RecordNotFound() {
		tag.Name = name
		tag.APIID = createAPIID()
		db.db.Model(user).Association("Tags").Append(&tag)
	}

	return tag
}

// Tags returns a list of all Tags owned by user
func (db *DB) Tags(user *models.User) (tags []models.Tag) {
	db.db.Model(user).Association("Tags").Find(&tags)
	return
}

// TagEntries with the given tag for user
func (db *DB) TagEntries(tagID string, entries []string, user *models.User) error {
	if len(entries) == 0 {
		return nil
	}

	tag, found := db.TagWithAPIID(tagID, user)
	if !found {
		return ErrModelNotFound
	}

	dbEntries := make([]models.Entry, len(entries))
	for i, entry := range entries {
		dbEntry, found := db.EntryWithAPIID(entry, user)
		if found {
			dbEntries[i] = dbEntry
		}
	}

	for _, entry := range dbEntries {
		db.db.Model(tag).Association("Entries").Append(&entry)
	}

	return nil
}

// EntriesFromTag returns all Entries which are tagged with tagID
func (db *DB) EntriesFromTag(tagID string, marker models.Marker, orderByNewest bool, user *models.User) (entries []models.Entry) {
	tag := &models.Tag{}
	if db.db.Model(user).Where("api_id = ?", tagID).Related(tag).RecordNotFound() {
		return
	}

	query := db.db.Model(tag)
	if marker != models.Any {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Association("Entries").Find(&entries)
	return
}

// EntriesFromMultipleTags returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func (db *DB) EntriesFromMultipleTags(tagIDs []string, orderByNewest bool, marker models.Marker, user *models.User) (entries []models.Entry) {
	order := db.db.Table("entries").Select("entries.title")
	if orderByNewest {
		order = order.Order("created_at DESC")
	} else {
		order = order.Order("created_at ASC")
	}

	if marker != models.Any {
		order = order.Where("mark = ?", marker)
	}

	var tagPrimaryKeys []uint
	for _, tag := range tagIDs {
		key := db.tagPrimaryKey(tag)
		if key != 0 {
			tagPrimaryKeys = append(tagPrimaryKeys, key)
		}
	}

	query := "inner join entry_tags ON entry_tags.entry_id = entries.id"
	order.Joins(query).Where("entry_tags.tag_id in (?)", tagPrimaryKeys).Scan(&entries)
	return
}

// TagPrimaryKey returns the SQL primary key of a Tag with an api_id
func (db *DB) tagPrimaryKey(apiID string) uint {
	tag := &models.Tag{}
	if db.db.First(tag, "api_id = ?", apiID).RecordNotFound() {
		return 0
	}
	return tag.ID
}

// EditTagName for the tag with the given API ID and owned by user
func (db *DB) EditTagName(tagID, name string, user *models.User) error {
	if tagInDB, found := db.TagWithAPIID(tagID, user); found {
		tagInDB.Name = name
		db.db.Model(tagInDB).Save(tagInDB)
		return nil
	}
	return ErrModelNotFound
}

// DeleteTag with id and owned by user
func (db *DB) DeleteTag(id string, user *models.User) error {
	if tag, found := db.TagWithAPIID(id, user); found {
		db.db.Delete(tag)
		return nil
	}
	return ErrModelNotFound
}

// CategoryStats returns all Stats for a Category with the given id and that is owned by user
func (db *DB) CategoryStats(id string, user *models.User) (stats models.Stats) {
	ctg := &models.Category{}
	if db.db.Model(user).Where("api_id = ?", id).Related(ctg).RecordNotFound() {
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
func (db *DB) FeedStats(id string, user *models.User) (stats models.Stats) {
	feed := &models.Feed{}
	if db.db.Model(user).Where("api_id = ?", id).Related(feed).RecordNotFound() {
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
	if feed, found := db.FeedWithAPIID(id, user); found {
		markedEntry := &models.Entry{Mark: marker}
		db.db.Model(markedEntry).Where("user_id = ? AND feed_id = ?", user.ID, feed.ID).Update(markedEntry)
		return nil
	}

	return ErrModelNotFound
}

// MarkCategory applies marker to a category with id and owned by user
func (db *DB) MarkCategory(id string, marker models.Marker, user *models.User) error {
	ctg, found := db.CategoryWithAPIID(id, user)
	if !found {
		return ErrModelNotFound
	}

	var feeds []models.Feed
	db.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	markedEntry := &models.Entry{Mark: marker}
	db.db.Model(markedEntry).Where("user_id = ? AND feed_id in (?)", user.ID, feedIds).Update(markedEntry)
	return nil
}

// MarkEntry applies marker to an entry with id and owned by user
func (db *DB) MarkEntry(id string, marker models.Marker, user *models.User) error {
	if entry, found := db.EntryWithAPIID(id, user); found {
		db.db.Model(&entry).Update(models.Entry{Mark: marker})
		return nil
	}
	return ErrModelNotFound
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
