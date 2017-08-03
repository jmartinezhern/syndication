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
	"crypto/rand"
	"encoding/base64"
	"io"
	mathRand "math/rand"
	"strconv"
	"time"

	"golang.org/x/crypto/scrypt"
)

const (
	pwSaltBytes = 32
	pwHashBytes = 64
)

var (
	lastTimeIDWasCreated int64
	random32Int          uint32
)

// CreatePasswordHashAndSalt for a given password
func CreatePasswordHashAndSalt(password string) ([]byte, []byte) {
	var err error

	salt := make([]byte, pwSaltBytes)
	_, err = io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err) // We must be able to read from random
	}

	hash, err := scrypt.Key([]byte(password), salt, 1<<14, 8, 1, pwHashBytes)
	if err != nil {
		panic(err) // We must never get an error
	}

	return hash, salt
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
func CreateAPIID() string {
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
