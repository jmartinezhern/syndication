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

package database

import (
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/varddum/syndication/models"
)

type (
	// A UserDB represents access to a SQL database in the context
	// of a specific user.
	UserDB struct {
		db   *gorm.DB
		user models.User
	}
)

func (udb *UserDB) createFeed(feed *models.Feed, ctg *models.Category) {
	feed.APIID = createAPIID()

	if ctg != nil {
		feed.Category = *ctg
		feed.CategoryID = ctg.ID
		feed.Category.APIID = ctg.APIID

		udb.db.Model(&udb.user).Association("Feeds").Append(feed)
		udb.db.Model(ctg).Association("Feeds").Append(feed)
	} else {
		udb.db.Model(&udb.user).Association("Feeds").Append(feed)
	}
}

// TagPrimaryKey returns the SQL primary key of a Tag with an api_id
func (udb *UserDB) tagPrimaryKey(apiID string) uint {
	tag := &models.Tag{}
	if udb.db.Model(&udb.user).Where("api_id = ?", apiID).Related(tag).RecordNotFound() {
		return 0
	}
	return tag.ID
}

// NewUserDB creates a new UserDB instance
func (db *DB) NewUserDB(user models.User) UserDB {
	return UserDB{
		db:   db.db,
		user: user,
	}
}

// CategoryWithName returns a Category that has a matching name and belongs to the given user
func (udb *UserDB) CategoryWithName(name string) (ctg models.Category, found bool) {
	found = !udb.db.Model(&udb.user).Where("name = ?", name).Related(&ctg).RecordNotFound()
	return
}

// TagWithName returns a Tag that has a matching name and belongs to the given user
func (udb *UserDB) TagWithName(name string) (tag models.Tag, found bool) {
	found = !udb.db.Model(&udb.user).Where("name = ?", name).Related(&tag).RecordNotFound()
	return
}

// EntryWithAPIID returns an Entry with id that belongs to user
func (udb *UserDB) EntryWithAPIID(apiID string) (entry models.Entry, found bool) {
	found = !udb.db.Model(&udb.user).Where("api_id = ?", apiID).Related(&entry).RecordNotFound()
	return
}

// TagWithAPIID returns a Tag with id that belongs to user
func (udb *UserDB) TagWithAPIID(apiID string) (tag models.Tag, found bool) {
	found = !udb.db.Model(&udb.user).Where("api_id = ?", apiID).Related(&tag).RecordNotFound()
	return
}

// NewAPIKey creates a new APIKey object owned by user
func (udb *UserDB) NewAPIKey(secret string, expiration time.Duration) (models.APIKey, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = udb.user.APIID
	claims["admin"] = false
	claims["exp"] = time.Now().Add(expiration).Unix()

	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return models.APIKey{}, err
	}

	key := &models.APIKey{
		Key:    t,
		User:   udb.user,
		UserID: udb.user.ID,
	}

	udb.db.Model(&udb.user).Association("APIKeys").Append(key)

	return *key, nil
}

// NewFeedWithCategory creates a new feed associated to a category with the given API ID
func (udb *UserDB) NewFeedWithCategory(title, subscription, ctgID string) (models.Feed, error) {
	ctg, found := udb.CategoryWithAPIID(ctgID)
	if !found {
		return models.Feed{}, ErrModelNotFound
	}

	feed := models.Feed{
		Title:        title,
		Subscription: subscription,
	}

	udb.createFeed(&feed, &ctg)

	return feed, nil
}

// NewFeed creates a new Feed object owned by user
func (udb *UserDB) NewFeed(title, subscription string) models.Feed {
	feed := models.Feed{
		Title:        title,
		Subscription: subscription,
	}

	ctg := models.Category{}
	udb.db.Model(&udb.user).Where("name = ?", models.Uncategorized).Related(&ctg)

	udb.createFeed(&feed, &ctg)

	return feed
}

// Feeds returns a list of all Feeds owned by a user
func (udb *UserDB) Feeds() (feeds []models.Feed) {
	udb.db.Model(&udb.user).Association("Feeds").Find(&feeds)
	return
}

// FeedsFromCategory returns all Feeds that belong to a category with categoryID
func (udb *UserDB) FeedsFromCategory(categoryID string) (feeds []models.Feed) {
	if ctg, found := udb.CategoryWithAPIID(categoryID); found {
		udb.db.Model(ctg).Association("Feeds").Find(&feeds)
	}
	return
}

// FeedWithAPIID returns a Feed with id and owned by user
func (udb *UserDB) FeedWithAPIID(id string) (models.Feed, bool) {
	feed := models.Feed{}

	if !udb.db.Model(&udb.user).Where("api_id = ?", id).Related(&feed).RecordNotFound() {
		udb.db.Model(&feed).Related(&feed.Category)
		return feed, true
	}

	return models.Feed{}, false
}

// DeleteFeed with id and owned by user
func (udb *UserDB) DeleteFeed(id string) error {
	foundFeed := &models.Feed{}
	if !udb.db.Model(&udb.user).Where("api_id = ?", id).Related(foundFeed).RecordNotFound() {
		udb.db.Delete(foundFeed)
		return nil
	}
	return ErrModelNotFound
}

// EditFeed owned by user
func (udb *UserDB) EditFeed(feed *models.Feed) error {
	if dbFeed, found := udb.FeedWithAPIID(feed.APIID); found {
		udb.db.Model(&dbFeed).Updates(feed)
		return nil
	}
	return ErrModelNotFound
}

// NewCategory creates a new Category object owned by user
func (udb *UserDB) NewCategory(name string) models.Category {
	ctg := models.Category{
		Name:  name,
		APIID: createAPIID(),
	}
	udb.db.Model(&udb.user).Association("Categories").Append(&ctg)
	return ctg
}

// EditCategory owned by user
func (udb *UserDB) EditCategory(ctg *models.Category) error {
	foundCtg := &models.Category{}
	if !udb.db.Model(&udb.user).Where("api_id = ?", ctg.APIID).Related(foundCtg).RecordNotFound() {
		foundCtg.Name = ctg.Name
		udb.db.Model(ctg).Save(foundCtg)
		return nil
	}
	return ErrModelNotFound
}

// DeleteCategory with id and owned by user
func (udb *UserDB) DeleteCategory(id string) error {
	ctg := &models.Category{}
	if udb.db.Model(&udb.user).Where("api_id = ?", id).Related(ctg).RecordNotFound() {
		return ErrModelNotFound
	}

	udb.db.Delete(ctg)
	return nil
}

// CategoryWithAPIID returns a Category with id and owned by user
func (udb *UserDB) CategoryWithAPIID(id string) (ctg models.Category, found bool) {
	found = !udb.db.Model(&udb.user).Where("api_id = ?", id).Related(&ctg).RecordNotFound()
	return
}

// Categories returns a list of all Categories owned by user
func (udb *UserDB) Categories() (categories []models.Category) {
	udb.db.Model(&udb.user).Association("Categories").Find(&categories)
	return
}

// ChangeFeedCategory changes the category a feed belongs to
func (udb *UserDB) ChangeFeedCategory(feedID string, ctgID string) error {
	feed := &models.Feed{}
	if udb.db.Model(&udb.user).Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return ErrModelNotFound
	}

	prevCtg := &models.Category{
		ID: feed.CategoryID,
	}

	udb.db.First(prevCtg)

	udb.db.Model(prevCtg).Association("Feeds").Delete(feed)

	newCtg := &models.Category{}
	if udb.db.Model(&udb.user).Where("api_id = ?", ctgID).Related(newCtg).RecordNotFound() {
		return ErrModelNotFound
	}

	udb.db.Model(newCtg).Association("Feeds").Append(feed)

	return nil
}

// NewEntry creates a new Entry object owned by user
func (udb *UserDB) NewEntry(entry models.Entry, feedID string) (models.Entry, error) {
	feed, found := udb.FeedWithAPIID(feedID)
	if !found {
		return models.Entry{}, ErrModelNotFound
	}

	entry.APIID = createAPIID()
	entry.Feed = feed
	entry.FeedID = feed.ID

	udb.db.Model(&udb.user).Association("Entries").Append(entry)
	udb.db.Model(&feed).Association("Entries").Append(entry)

	return entry, nil
}

// NewEntries creates multiple new Entry objects which
// are all owned by feed with feedAPIID and user
func (udb *UserDB) NewEntries(entries []models.Entry, feedID string) ([]models.Entry, error) {
	if len(entries) == 0 {
		// Nothing to do
		return nil, nil
	}

	feed, found := udb.FeedWithAPIID(feedID)
	if !found {
		return nil, ErrModelNotFound
	}

	for i, entry := range entries {
		entry.APIID = createAPIID()

		udb.db.Model(&udb.user).Association("Entries").Append(&entry)
		udb.db.Model(&feed).Association("Entries").Append(&entry)

		entries[i] = entry
	}

	return entries, nil
}

// EntryWithGUIDExists returns true if an Entry exists with the given guid and is owned by user
func (udb *UserDB) EntryWithGUIDExists(guid string, feedID string) bool {
	userModel := udb.db.Model(&udb.user)
	feed := new(models.Feed)
	if userModel.Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return false
	}

	return !userModel.Where("guid = ? AND feed_id = ?", guid, feed.ID).Related(&models.Entry{}).RecordNotFound()
}

// Entries returns a list of all entries owned by user
func (udb *UserDB) Entries(orderByNewest bool, marker models.Marker) []models.Entry {
	entries := []models.Entry{}
	if marker == models.MarkerNone {
		return nil
	}

	query := udb.db.Model(&udb.user)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Association("Entries").Find(&entries)

	return entries
}

// EntriesFromFeed returns all Entries that belong to a feed with feedID
func (udb *UserDB) EntriesFromFeed(feedID string, orderByNewest bool, marker models.Marker) []models.Entry {
	if marker == models.MarkerNone {
		return nil
	}

	feed := &models.Feed{}
	if udb.db.Model(&udb.user).Where("api_id = ?", feedID).Related(feed).RecordNotFound() {
		return nil
	}

	entries := []models.Entry{}

	query := udb.db.Model(&udb.user)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	query.Where("feed_id = ?", feed.ID).Association("Entries").Find(&entries)

	return entries
}

// EntriesFromCategory returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func (udb *UserDB) EntriesFromCategory(categoryID string, orderByNewest bool, marker models.Marker) []models.Entry {
	if marker == models.MarkerNone {
		return nil
	}

	category := &models.Category{}
	if udb.db.Model(&udb.user).Where("api_id = ?", categoryID).Related(category).RecordNotFound() {
		return nil
	}

	var feeds []models.Feed
	var entries []models.Entry

	udb.db.Model(category).Related(&feeds)

	query := udb.db.Model(&udb.user)
	if marker != models.MarkerAny {
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

	return entries
}

// NewTag creates a new Tag object owned by user
func (udb *UserDB) NewTag(name string) models.Tag {
	tag := models.Tag{}
	if udb.db.Model(&udb.user).Where("name = ?", name).Related(&tag).RecordNotFound() {
		tag.Name = name
		tag.APIID = createAPIID()
		udb.db.Model(&udb.user).Association("Tags").Append(&tag)
	}

	return tag
}

// Tags returns a list of all Tags owned by user
func (udb *UserDB) Tags() (tags []models.Tag) {
	udb.db.Model(&udb.user).Association("Tags").Find(&tags)
	return
}

// TagEntries with the given tag for user
func (udb *UserDB) TagEntries(tagID string, entries []string) error {
	if len(entries) == 0 {
		return nil
	}

	tag, found := udb.TagWithAPIID(tagID)
	if !found {
		return ErrModelNotFound
	}

	dbEntries := make([]models.Entry, len(entries))
	for i, entry := range entries {
		dbEntry, found := udb.EntryWithAPIID(entry)
		if found {
			dbEntries[i] = dbEntry
		}
	}

	for _, entry := range dbEntries {
		udb.db.Model(tag).Association("Entries").Append(&entry)
	}

	return nil
}

// EntriesFromTag returns all Entries which are tagged with tagID
func (udb *UserDB) EntriesFromTag(tagID string, marker models.Marker, orderByNewest bool) []models.Entry {
	tag := &models.Tag{}
	if udb.db.Model(&udb.user).Where("api_id = ?", tagID).Related(tag).RecordNotFound() {
		return nil
	}

	query := udb.db.Model(tag)
	if marker != models.MarkerAny {
		query = query.Where("mark = ?", marker)
	}

	if orderByNewest {
		query = query.Order("published DESC")
	} else {
		query = query.Order("published ASC")
	}

	var entries []models.Entry

	query.Association("Entries").Find(&entries)

	return entries
}

// EntriesFromMultipleTags returns all Entries that are related to a Category with categoryID by the entries' owning Feed
func (udb *UserDB) EntriesFromMultipleTags(tagIDs []string, orderByNewest bool, marker models.Marker) []models.Entry {
	order := udb.db.Model(&udb.user).Select("entries.title")
	if orderByNewest {
		order = order.Order("created_at DESC")
	} else {
		order = order.Order("created_at ASC")
	}

	if marker != models.MarkerAny {
		order = order.Where("mark = ?", marker)
	}

	var tagPrimaryKeys []uint
	for _, tag := range tagIDs {
		key := udb.tagPrimaryKey(tag)
		if key != 0 {
			tagPrimaryKeys = append(tagPrimaryKeys, key)
		}
	}

	var entries []models.Entry

	query := "inner join entry_tags ON entry_tags.entry_id = entries.id"
	order.Joins(query).Where("entry_tags.tag_id in (?)", tagPrimaryKeys).Related(&entries)

	return entries
}

// EditTagName for the tag with the given API ID and owned by user
func (udb *UserDB) EditTagName(tagID, name string) error {
	if tagInDB, found := udb.TagWithAPIID(tagID); found {
		tagInDB.Name = name
		udb.db.Model(tagInDB).Save(tagInDB)
		return nil
	}
	return ErrModelNotFound
}

// DeleteTag with id and owned by user
func (udb *UserDB) DeleteTag(id string) error {
	if tag, found := udb.TagWithAPIID(id); found {
		udb.db.Delete(tag)
		return nil
	}
	return ErrModelNotFound
}

// CategoryStats returns all Stats for a Category with the given id and that is owned by user
func (udb *UserDB) CategoryStats(id string) models.Stats {
	ctg := &models.Category{}
	if udb.db.Model(&udb.user).Where("api_id = ?", id).Related(ctg).RecordNotFound() {
		return models.Stats{}
	}

	var feeds []models.Feed
	udb.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	query := udb.db.Model(&udb.user).Where("feed_id in (?)", feedIds)

	stats := models.Stats{}

	stats.Unread = query.Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = query.Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = query.Where("saved = ?", true).Association("Entries").Count()
	stats.Total = query.Association("Entries").Count()

	return stats
}

// FeedStats returns all Stats for a Feed with the given id and that is owned by user
func (udb *UserDB) FeedStats(id string) models.Stats {
	feed := &models.Feed{}
	if udb.db.Model(&udb.user).Where("api_id = ?", id).Related(feed).RecordNotFound() {
		return models.Stats{}
	}

	stats := models.Stats{}

	stats.Unread = udb.db.Model(&udb.user).Where("feed_id = ? AND mark = ?", feed.ID, models.MarkerUnread).Association("Entries").Count()
	stats.Read = udb.db.Model(&udb.user).Where("feed_id = ? AND mark = ?", feed.ID, models.MarkerRead).Association("Entries").Count()
	stats.Saved = udb.db.Model(&udb.user).Where("feed_id = ? AND saved = ?", feed.ID, true).Association("Entries").Count()
	stats.Total = udb.db.Model(&udb.user).Where("feed_id = ?", feed.ID).Association("Entries").Count()

	return stats
}

// Stats returns all Stats for the given user
func (udb *UserDB) Stats() models.Stats {
	stats := models.Stats{}

	stats.Unread = udb.db.Model(&udb.user).Where("mark = ?", models.MarkerUnread).Association("Entries").Count()
	stats.Read = udb.db.Model(&udb.user).Where("mark = ?", models.MarkerRead).Association("Entries").Count()
	stats.Saved = udb.db.Model(&udb.user).Where("saved = ?", true).Association("Entries").Count()
	stats.Total = udb.db.Model(&udb.user).Association("Entries").Count()

	return stats
}

// MarkFeed applies marker to a Feed with id and owned by user
func (udb *UserDB) MarkFeed(id string, marker models.Marker) error {
	if feed, found := udb.FeedWithAPIID(id); found {
		markedEntry := &models.Entry{Mark: marker}
		udb.db.Model(markedEntry).Where("user_id = ? AND feed_id = ?", udb.user.ID, feed.ID).Update(markedEntry)
		return nil
	}

	return ErrModelNotFound
}

// MarkCategory applies marker to a category with id and owned by user
func (udb *UserDB) MarkCategory(id string, marker models.Marker) error {
	ctg, found := udb.CategoryWithAPIID(id)
	if !found {
		return ErrModelNotFound
	}

	var feeds []models.Feed
	udb.db.Model(&ctg).Association("Feeds").Find(&feeds)

	feedIds := make([]uint, len(feeds))
	for i, feed := range feeds {
		feedIds[i] = feed.ID
	}

	markedEntry := &models.Entry{Mark: marker}
	udb.db.Model(markedEntry).Where("user_id = ? AND feed_id in (?)", udb.user.ID, feedIds).Update(markedEntry)
	return nil
}

// MarkEntry applies marker to an entry with id and owned by user
func (udb *UserDB) MarkEntry(id string, marker models.Marker) error {
	if entry, found := udb.EntryWithAPIID(id); found {
		udb.db.Model(&entry).Update(models.Entry{Mark: marker})
		return nil
	}
	return ErrModelNotFound
}

// KeyBelongsToUser returns true if the given APIKey is owned by user
func (udb *UserDB) KeyBelongsToUser(key models.APIKey) bool {
	return !udb.db.Model(&udb.user).Where("key = ?", key.Key).Related(&key).RecordNotFound()
}
