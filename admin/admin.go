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

//TODO: Consider SO_PEERCRED with Unix Sockets

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"reflect"
	"sync"

	"github.com/chavamee/syndication/database"
	log "github.com/sirupsen/logrus"
)

type (
	// Admin contains all necessary resources for an administration
	// api. This includes unix connection resources and command handler
	// information.
	Admin struct {
		ln          *net.UnixListener
		socketPath  string
		State       chan state
		db          *database.DB
		lock        sync.Mutex
		cmdHandlers map[string]reflect.Value
		connections []*net.UnixConn
	}

	// Request represents a request made to the Admin API
	Request struct {
		Command   string `json:"command"`
		Arguments args   `json:"arguments"`
	}

	// Response represents a response given as a result
	// of a request to the Admin API.
	Response struct {
		Status StatusCode  `json:"status"`
		Error  string      `json:"error,omitempty"`
		Result interface{} `json:"result,optional"`
	}
)

// StatusCode represents the status of a request.
type StatusCode int

const (
	// OK signals that the command was successful.
	OK StatusCode = iota

	// NotImplemented signals that the command requested
	// is not implemented by the service.
	NotImplemented

	// UnkownCommand signals that the requested command
	// could not be identified.
	UnkownCommand

	// BadRequest signals that the request is invalid.
	BadRequest

	// BadArgument signals that the some or all of the
	// given arguments are invalid.
	BadArgument

	// DatabaseError signals that the command failed due to
	// a database error.
	DatabaseError

	// InternalError signals that the command failed due to
	// other errors.
	InternalError
)

type args map[string]interface{}

type state int

const (
	started state = iota
	listening
	stopping
	stopped
)

const defaultSocketPath = "/var/run/syndication/admin"

// NewUser creates a user
func (a *Admin) NewUser(args args, r *Response) error {
	var username string
	var password string

	r.Status = BadArgument

	aVal := reflect.ValueOf(args["username"])
	if aVal.Kind() != reflect.String {
		r.Error = "Bad first argument"
		return nil
	}

	bVal := reflect.ValueOf(args["password"])
	if bVal.Kind() != reflect.String {
		r.Error = "Bad second argument"
		return nil
	}

	username = aVal.String()
	password = bVal.String()

	if err := a.db.NewUser(username, password); err != nil {
		dbError := err.(database.DBError)
		r.Status = DatabaseError
		r.Error = dbError.Error()
		return nil
	}

	r.Status = OK
	r.Error = "OK"
	return nil
}

// DeleteUser deletes a user
func (a *Admin) DeleteUser(args args, r *Response) error {
	var userID string

	r.Status = BadArgument
	r.Error = "Bad first argument"

	aVal := reflect.ValueOf(args["userID"])
	if aVal.Kind() != reflect.String {
		return nil
	}

	userID = aVal.String()

	err := a.db.DeleteUser(userID)
	if err != nil {
		dbError := err.(database.DBError)
		r.Status = DatabaseError
		r.Error = dbError.Error()
		return nil
	}

	r.Status = OK
	r.Error = "OK"

	return nil
}

// ChangeUserName changes a user's name
func (a *Admin) ChangeUserName(args args, r *Response) error {
	var userID string
	var newName string

	r.Status = BadArgument

	aVal := reflect.ValueOf(args["userID"])
	if aVal.Kind() != reflect.String {
		r.Error = "Bad first argument"
		return nil
	}

	bVal := reflect.ValueOf(args["newName"])
	if bVal.Kind() != reflect.String {
		r.Error = "Bad second argument"
		return nil
	}

	userID = aVal.String()
	newName = bVal.String()

	err := a.db.ChangeUserName(userID, newName)
	if err != nil {
		dbError := err.(database.DBError)
		r.Status = DatabaseError
		r.Error = dbError.Error()
		return nil
	}

	r.Status = OK
	r.Error = "OK"

	return nil
}

// ChangeUserPassword changes a user's password.
func (a *Admin) ChangeUserPassword(args args, r *Response) error {
	var userID string
	var newPassword string

	r.Status = BadArgument

	aVal := reflect.ValueOf(args["userID"])
	if aVal.Kind() != reflect.String {
		r.Error = "Bad first argument"
		return nil
	}

	bVal := reflect.ValueOf(args["newPassword"])
	if bVal.Kind() != reflect.String {
		r.Error = "Bad second argument"
		return nil
	}

	userID = aVal.String()
	newPassword = bVal.String()

	err := a.db.ChangeUserPassword(userID, newPassword)
	if err != nil {
		dbError := err.(database.DBError)
		r.Status = DatabaseError
		r.Error = dbError.Error()
		return nil
	}

	r.Status = OK
	r.Error = "OK"

	return nil
}

// GetUsers returns a list of all existing users.
func (a *Admin) GetUsers(args args, r *Response) error {
	r.Status = OK
	r.Error = "OK"

	r.Result = a.db.Users("id,created_at,updated_at,uuid,email,username")

	return nil
}

// GetUser returns all information on a user.
func (a *Admin) GetUser(args args, r *Response) error {
	var userID string

	r.Status = OK

	aVal := reflect.ValueOf(args["userID"])
	if aVal.Kind() != reflect.String {
		r.Error = "Bad first argument"
		return nil
	}

	userID = aVal.String()

	user, err := a.db.UserWithUUID(userID)
	if err != nil {
		dbError := err.(database.DBError)
		r.Status = DatabaseError
		r.Error = dbError.Error()
		return nil
	}

	r.Result = user
	r.Status = OK
	r.Error = "OK"

	return nil
}

// NewAdmin creates a new Admin socket and initializes administration handlers
func NewAdmin(db *database.DB, socketPath string) (a *Admin, err error) {
	a = &Admin{
		db:    db,
		State: make(chan state),
	}

	if socketPath != "" {
		a.socketPath = socketPath
	} else {
		a.socketPath = defaultSocketPath
	}

	if _, err = os.Stat(a.socketPath); err == nil {
		err = os.Remove(a.socketPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	a.ln, err = net.ListenUnix("unixpacket", &net.UnixAddr{
		Name: a.socketPath,
		Net:  "unixpacket"})

	if err != nil {
		log.Fatal(err)
	}

	aVal := reflect.ValueOf(a)
	a.cmdHandlers = map[string]reflect.Value{
		"NewUser":            aVal.MethodByName("NewUser"),
		"DeleteUser":         aVal.MethodByName("DeleteUser"),
		"GetUsers":           aVal.MethodByName("GetUsers"),
		"GetUser":            aVal.MethodByName("GetUser"),
		"ChangeUserName":     aVal.MethodByName("ChangeUserName"),
		"ChangeUserPassword": aVal.MethodByName("ChangeUserPassword"),
	}

	return
}

// Start listening at the administration socket
func (a *Admin) Start() {
	go a.listen()
	a.State <- listening
}

// Stop listening at the administration socket
// and optionally wait for a full stop.
func (a *Admin) Stop(wait bool) {
	err := a.ln.Close()
	if err != nil {
		log.Error(err)
	}
	a.State <- stopping

	if wait {
		<-a.State
	}

	if _, err := os.Stat(a.socketPath); err == nil {
		err = os.Remove(a.socketPath)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (a *Admin) listen() {
	shouldStop := false
	for !shouldStop {
		switch <-a.State {
		case listening:
			conn, err := a.ln.AcceptUnix()
			if err == nil {
				go a.handleConnection(conn)
			}

			break
		case stopping:
			for _, conn := range a.connections {
				conn.Close()
			}

			shouldStop = true
			break
		}
	}

	a.State <- stopped
}

func (a *Admin) handleConnection(conn *net.UnixConn) {
	if conn == nil {
		log.Warn("Connection could not be accepted")
		return
	}

	a.connections = append(a.connections, conn)

	a.State <- listening

	for {
		req := &Request{}
		resp := &Response{}
		err := json.NewDecoder(conn).Decode(req)

		// DB blocks on an operation on it but we should not rely on it.
		if err == nil {
			a.lock.Lock()

			err = a.processRequest(*req, resp)
			if err != nil {
				resp.Status = InternalError
				resp.Error = "Failed to process request"
				log.Error(err)
			}

			a.lock.Unlock()
		} else if err == io.EOF {
			return
		} else {
			resp.Status = BadRequest
			resp.Error = "Request is not valid JSON"
		}

		err = json.NewEncoder(conn).Encode(resp)
		if err != nil {
			log.Error(err)
		}
	}
}

func (a *Admin) processRequest(req Request, resp *Response) error {
	method := a.cmdHandlers[req.Command]
	if !method.IsValid() {
		resp.Status = NotImplemented
		resp.Error = req.Command + " is not implemented."
		return nil
	}

	for key, val := range req.Arguments {
		if !reflect.ValueOf(val).IsValid() {
			resp.Status = BadArgument
			resp.Error = "Argument " + key + " is invalid."
			return nil
		}
	}

	args := []reflect.Value{reflect.ValueOf(req.Arguments), reflect.ValueOf(resp)}
	rtVals := method.Call(args)
	if !rtVals[0].IsNil() {
		return rtVals[0].Interface().(error)
	}

	return nil
}
