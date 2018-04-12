// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"

	"github.com/varddum/syndication/models"
	"github.com/varddum/syndication/plugins"
)

func helloWorldHandler(c plugins.APICtx, w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "Hello, %q\n", html.EscapeString(r.URL.Path))
	if err != nil {
		fmt.Println(err)
	}
}

func entriesHandler(c plugins.APICtx, w http.ResponseWriter, r *http.Request) {
	if c.HasUser() {
		entries := c.User.Entries(true, models.MarkerAny)
		if len(entries) == 0 {
			fmt.Fprintf(w, "Nothing new!\n")
			return
		}

		js, err := json.Marshal(entries)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(js)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// Initialize creates and registers this plugins information and available endpoints
func Initialize() (plugins.Plugin, error) {
	plugin := plugins.NewAPIPlugin("API Example")

	err := plugin.RegisterEndpoint(plugins.Endpoint{
		NeedsUser: false,
		Path:      "/hello_world",
		Method:    "GET",
		Group:     "api_test",
		Handler:   helloWorldHandler,
	})

	if err != nil {
		return plugin, err
	}

	err = plugin.RegisterEndpoint(plugins.Endpoint{
		NeedsUser: true,
		Path:      "/entries",
		Method:    "GET",
		Group:     "api_test",
		Handler:   entriesHandler,
	})

	return plugin, err
}

// Shutdown and cleanup the plugin
func Shutdown() {
	fmt.Println("Shutting down hello_world plugin.")
}
