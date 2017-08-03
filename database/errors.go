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

// DBError is the error type returned when any database operation
// does not return expected result or cannot be carried out.
type DBError interface {
	String() string
	Code() int
	Error() string
}

type (
	// Conflict is a DBError returned when a database operation
	// cannot be carried out because it conflicts with a previous operation.
	Conflict struct {
		msg string
	}

	// NotFound is DBError returned when an object cannot be found in the
	// database.
	NotFound struct {
		msg string
	}

	// BadRequest is a DBError returned when an operation is malformed.
	BadRequest struct {
		msg string
	}

	// Unauthorized is a DBError returned when a client does not have the permissions
	// to carry out an operation
	Unauthorized struct {
		msg string
	}
)

func (e Conflict) Error() string {
	return e.msg
}

func (e Conflict) String() string {
	return "Conflict"
}

// Code returns Conflict's corresponding error code
func (e Conflict) Code() int {
	return 409
}

func (e NotFound) Error() string {
	return e.msg
}

func (e NotFound) String() string {
	return "NotFound"
}

// Code returns NotFound's corresponding error code
func (e NotFound) Code() int {
	return 404
}

func (e BadRequest) Error() string {
	return e.msg
}

func (e BadRequest) String() string {
	return "BadRequest"
}

// Code returns BadRequest's corresponding error code
func (e BadRequest) Code() int {
	return 400
}

func (e Unauthorized) Error() string {
	return e.msg
}

func (e Unauthorized) String() string {
	return "Unauthorized"
}

// Code returns Unauthorized's corresponding error code
func (e Unauthorized) Code() int {
	return 401
}
