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

package admin

import (
	"errors"
	"net"
	"net/rpc"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/varddum/syndication/database"
)

type (
	// NewUserArgs collects arguments for the
	// NewUser routine
	NewUserArgs struct {
		Username, Password string
	}

	// ChangeUserNameArgs collects arguments for the
	// ChangeUserName routine
	ChangeUserNameArgs struct {
		UserID, NewName string
	}

	// ChangeUserPasswordArgs collects arguments for the
	// ChangeUserPassword routine
	ChangeUserPasswordArgs struct {
		UserID, NewPassword string
	}

	// User collects user information that can be exported by
	// the rpc service
	User struct {
		Name, ID string
	}

	// Service represents a rpc service administration
	// routines
	Service struct {
		ln         *net.UnixListener
		socketPath string
		state      chan bool
		server     *rpc.Server
	}

	// Admin represents adminstration routines available over rpc
	Admin struct {
	}
)

const defaultSocketPath = "/var/run/syndication/admin"

// NewUser creates a new user
func (a *Admin) NewUser(args NewUserArgs, msg *string) error {
	if _, found := database.UserWithName(args.Username); found {
		*msg = "User already exists"
		return errors.New("User already exists")
	}

	database.NewUser(args.Username, args.Password)

	return nil
}

// DeleteUser deletes a user with userID
func (a *Admin) DeleteUser(userID string, msg *string) error {
	return database.DeleteUser(userID)
}

// GetUserID retrieves the user id for a user with username
func (a *Admin) GetUserID(username string, userID *string) error {
	if user, found := database.UserWithName(username); found {
		*userID = user.APIID
		return nil
	}

	return errors.New("User does not exist")
}

// GetUsers will return all existing usernames with their associated IDs
func (a *Admin) GetUsers(outLen int, users *[]User) error {
	dbUsers := database.Users("username,id")
	*users = make([]User, outLen)
	for idx, user := range dbUsers {
		if idx >= outLen {
			break
		}

		(*users)[idx] = User{user.Username, user.APIID}
	}

	return nil
}

// ChangeUserName modifies the username for a user with userID
func (a *Admin) ChangeUserName(args ChangeUserNameArgs, msg *string) error {
	return database.ChangeUserName(args.UserID, args.NewName)
}

// ChangeUserPassword modifies the password for a user with userID
func (a *Admin) ChangeUserPassword(args ChangeUserPasswordArgs, msg *string) error {
	return database.ChangeUserPassword(args.UserID, args.NewPassword)
}

// Start a admin rpc service
func (s *Service) Start() {
	go func() {
		s.server.Accept(s.ln)
		s.state <- true
	}()
}

// Stop an admin rpc service
func (s *Service) Stop() {
	if err := s.ln.Close(); err != nil {
		log.Error(err)
	}
	<-s.state
}

// NewService creates a new admin rpc service
func NewService(socketPath string) (*Service, error) {
	s := &Service{
		server: rpc.NewServer(),
		state:  make(chan bool),
	}

	if socketPath != "" {
		s.socketPath = socketPath
	} else {
		s.socketPath = defaultSocketPath
	}

	_, err := os.Stat(s.socketPath)
	if err == nil {
		err = os.Remove(s.socketPath)
		if err != nil {
			return nil, err
		}
	}

	s.ln, err = net.ListenUnix("unixpacket", &net.UnixAddr{
		Name: s.socketPath,
		Net:  "unixpacket"})

	if err != nil {
		return nil, err
	}

	a := &Admin{}

	err = s.server.Register(a)

	return s, err
}
