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
	"net"
	"net/rpc"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/varddum/syndication/usecases"
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
		adminUsecase usecases.Admin
	}
)

const defaultSocketPath = "/var/run/syndication/admin"

// NewUser creates a new user
func (a *Admin) NewUser(args NewUserArgs, msg *string) error {
	_, err := a.adminUsecase.NewUser(args.Username, args.Password)
	if err != nil {
		*msg = err.Error()
		return err
	}

	return nil
}

// DeleteUser deletes a user with userID
func (a *Admin) DeleteUser(userID string, msg *string) error {
	return a.adminUsecase.DeleteUser(userID)
}

// GetUsers will return all existing usernames with their associated IDs
func (a *Admin) GetUsers(outLen int, users *[]User) error {
	dbUsers := a.adminUsecase.Users()
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
	return a.adminUsecase.ChangeUserName(args.UserID, args.NewName)
}

// ChangeUserPassword modifies the password for a user with userID
func (a *Admin) ChangeUserPassword(args ChangeUserPasswordArgs, msg *string) error {
	return a.adminUsecase.ChangeUserPassword(args.UserID, args.NewPassword)
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

	a := &Admin{adminUsecase: &usecases.AdminUsecase{}}

	err = s.server.Register(a)

	return s, err
}
