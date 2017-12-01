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

package plugins

import (
	"net/http"
	"plugin"

	log "github.com/sirupsen/logrus"
)

// RequestHandler represents the function type for Endpoint Handlers in API Plugins
type RequestHandler = func(APICtx, http.ResponseWriter, *http.Request)

// InitFunc is a type alias for a plugin's initialization function
type InitFunc = func() (Plugin, error)

// ShutdownFunc is a type alias for a plugin's shutdown function
type ShutdownFunc = func()

type (
	// A Plugin is a linkable software component that registers to our Plugins
	Plugin interface {
		// Path to the plugin's shared object
		Path() string

		// Name taken by the plugin
		Name() string
	}

	// Endpoint represents an API Endpoint that can be registered by an API Plugin.
	Endpoint struct {
		NeedsUser bool
		Path      string
		Method    string
		Group     string
		Handler   RequestHandler
	}

	// APIPlugin collects information on an API Plugin and the endpoints it registers.
	APIPlugin struct {
		shutdownHandler ShutdownFunc
		name            string
		endpoints       []Endpoint
		path            string
	}

	// Plugins manages the available plugins configured and registered for a Syndication instance.
	Plugins struct {
		apiPlugins []APIPlugin
	}

	// APIPluginError represents an error that occurred in loading or acting on a plugin.
	APIPluginError struct {
		ErrorMsg string
	}
)

func (e APIPluginError) Error() string {
	return e.ErrorMsg
}

// Path return the plugin's location in the file system
func (p APIPlugin) Path() string {
	return p.path
}

// Name returns the plugin's registered name
func (p APIPlugin) Name() string {
	return p.name
}

// NewAPIPlugin creates a new API Plugin
func NewAPIPlugin(name string) APIPlugin {
	return APIPlugin{name: name}
}

// Endpoints returns all API endpoints registered by an API Plugin
func (p *APIPlugin) Endpoints() []Endpoint {
	return p.endpoints
}

// Shutdown calls a plugin's shutdown function
func (p *APIPlugin) Shutdown() {
	p.shutdownHandler()
}

// RegisterEndpoint appends an API Endpoint to the plugin's list of Endpoints.
func (p *APIPlugin) RegisterEndpoint(endpnt Endpoint) error {
	if endpnt.Handler == nil {
		return APIPluginError{"A handler is required."}
	}

	if endpnt.Method == "" {
		return APIPluginError{"A method is required."}
	}

	if endpnt.Path == "" {
		return APIPluginError{"A path is required."}
	}

	if p.checkConflictingPaths(endpnt) {
		return APIPluginError{"The path " + endpnt.Path + "for method " + endpnt.Method + " already exists."}
	}

	p.endpoints = append(p.endpoints, endpnt)

	return nil
}

func (p APIPlugin) checkConflictingPaths(incomingEndpnt Endpoint) bool {
	// TODO: This will be a linear search for now.
	for _, endpnt := range p.endpoints {
		if endpnt.Path == incomingEndpnt.Path && endpnt.Method == incomingEndpnt.Method {
			return true
		}
	}

	return false
}

// NewPlugins creates a new Plugins representation which can be used to load and verify
// plugins.
func NewPlugins(pluginPaths []string) Plugins {
	plugins := Plugins{}

	plugins.loadPlugins(pluginPaths)

	return plugins
}

func (s *Plugins) loadPlugins(paths []string) {
	for _, path := range paths {
		plgn, err := plugin.Open(path)
		if err != nil {
			log.Error(err, ". Skipping.")
			continue
		}

		initFuncSymb, err := plgn.Lookup("Initialize")
		if err != nil {
			log.Error(err, ". Skipping.")
			continue
		}

		initFunc, ok := initFuncSymb.(InitFunc)
		if !ok {
			log.Error("Invalid Initialization function.")
			continue
		}

		incomingPlgn, err := initFunc()
		if err != nil {
			log.Error(err, ". Skpping.")
			continue
		}

		switch t := incomingPlgn.(type) {
		case APIPlugin:
			shutdownFuncSymb, err := plgn.Lookup("Shutdown")
			if err != nil {
				log.Error(err, ". Skpping.")
				continue
			}

			shutdownFunc, ok := shutdownFuncSymb.(ShutdownFunc)
			if !ok {
				log.Error("Invalid Shutdown function.")
				continue
			}

			t.shutdownHandler = shutdownFunc

			s.apiPlugins = append(s.apiPlugins, t)

		default:
			log.Error("Unrecognized plugin type.")
		}

	}
}

// APIPlugins returns all API plugins that were loaded successfully
func (s *Plugins) APIPlugins() []APIPlugin {
	return s.apiPlugins
}
