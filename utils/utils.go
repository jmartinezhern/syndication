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

// Package utils provides utilities for other packages
package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	mathRand "math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mmcdole/gofeed"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/scrypt"

	"github.com/jmartinezhern/syndication/models"
)

const (
	pwSaltBytes = 32
	pwHashBytes = 64
)

var (
	lastTimeIDWasCreated int64
	random32Int          uint32
)

const (
	refreshKeyExpirationInterval = time.Hour * 24 * 7
	accessKeyExpirationInterval  = time.Hour * 24 * 3
)

var (
// ErrParsingFeed Signals that an error occurred while processing
// a RSS or Atom feed
)

func fetchFeed(url, etag string) (gofeed.Feed, error) {
	client := &http.Client{
		CheckRedirect: func(r *http.Request, v []*http.Request) error { return http.ErrUseLastResponse },
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return gofeed.Feed{}, err
	}

	req.Header.Add("If-None-Match", etag)

	resp, err := client.Do(req)
	if err != nil {
		return gofeed.Feed{}, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Warn(err)
		}
	}()

	fetchedFeed, err := gofeed.NewParser().Parse(resp.Body)
	if err != nil {
		return gofeed.Feed{}, err
	}

	return *fetchedFeed, nil
}

func convertItemToEntry(item *gofeed.Item) models.Entry {
	entry := models.Entry{
		Title: item.Title,
		Link:  item.Link,
		Mark:  models.MarkerUnread,
	}

	if item.Author != nil {
		entry.Author = item.Author.Name
	}

	if item.PublishedParsed != nil {
		entry.Published = *item.PublishedParsed
	} else {
		entry.Published = time.Now()
	}

	if item.GUID != "" {
		entry.GUID = item.GUID
	} else {
		itemHash := md5.Sum([]byte(item.Title + item.Link))
		entry.GUID = string(itemHash[:md5.Size])
	}

	return entry
}

// PullFeed and return all entries for that feed. If getting the
// subscription source or parsing the response fails, this function
// will error.
func PullFeed(url, etag string) (models.Feed, []models.Entry, error) {
	fetchedFeed, err := fetchFeed(url, etag)
	if err != nil {
		return models.Feed{}, nil, err
	}

	feed := models.Feed{
		Title:       fetchedFeed.Title,
		Description: fetchedFeed.Description,
		Source:      fetchedFeed.Link,
		LastUpdated: time.Now(),
	}

	entries := make([]models.Entry, len(fetchedFeed.Items))
	for idx, item := range fetchedFeed.Items {
		entries[idx] = convertItemToEntry(item)
	}

	return feed, entries, nil
}

// CreatePasswordHashAndSalt for a given password
func CreatePasswordHashAndSalt(password string) (hash, salt []byte) {
	var err error

	salt = make([]byte, pwSaltBytes)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err) // We must be able to read from random
	}

	hash, err = scrypt.Key([]byte(password), salt, 1<<14, 8, 1, pwHashBytes)
	if err != nil {
		panic(err) // We must never get an error
	}

	return
}

// VerifyPasswordHash with a given salt
func VerifyPasswordHash(password string, pwHash, pwSalt []byte) bool {
	hash, err := scrypt.Key([]byte(password), pwSalt, 1<<14, 8, 1, pwHashBytes)
	if err != nil {
		return false
	}

	if len(pwHash) != len(hash) {
		return false
	}

	for i, hashByte := range hash {
		if hashByte != pwHash[i] {
			return false
		}
	}

	return true
}

// CreateAPIID creates an API ID
func CreateID() string {
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

func NewAPIKey(secret string, keyType models.APIKeyType, userID string) (models.APIKey, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = userID

	switch keyType {
	case models.RefreshKey:
		claims["exp"] = time.Now().Add(refreshKeyExpirationInterval).Unix()
		claims["type"] = "refresh"
	case models.AccessKey:
		claims["exp"] = time.Now().Add(accessKeyExpirationInterval).Unix()
		claims["type"] = "access"
	}

	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return models.APIKey{}, err
	}

	return models.APIKey{
		Key:  t,
		Type: keyType,
	}, nil
}

func NewKeyPair(secret, userID string) (models.APIKeyPair, error) {
	accessKey, err := NewAPIKey(secret, models.AccessKey, userID)
	if err != nil {
		return models.APIKeyPair{}, err
	}

	refreshKey, err := NewAPIKey(secret, models.RefreshKey, userID)
	if err != nil {
		return models.APIKeyPair{}, err
	}

	return models.APIKeyPair{
		AccessKey:  accessKey.Key,
		RefreshKey: refreshKey.Key,
	}, nil
}

func ParseJWTClaims(secret, signingMethod, token string) (jwt.MapClaims, error) {
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		// Check the signing method
		if t.Method.Alg() != signingMethod {
			return nil, errors.New("jwt signing methods mismatch")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	return jwtToken.Claims.(jwt.MapClaims), nil
}
