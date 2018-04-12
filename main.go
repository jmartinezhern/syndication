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

package main

import (
	"os"
	"os/signal"

	"github.com/fatih/color"
	"github.com/urfave/cli"

	"github.com/varddum/syndication/admin"
	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/plugins"
	"github.com/varddum/syndication/server"
	"github.com/varddum/syndication/sync"
)

var intSignal chan os.Signal

const appName = "syndication"
const appUsage = "An flexible RSS server"

var appFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "config",
		Usage: "Path to a configuration file",
	},
	cli.StringFlag{
		Name:  "socket",
		Usage: "Path to admin socket",
	},
	cli.BoolFlag{
		Name:  "admin",
		Usage: "Enable/Disable admin",
	},
	cli.BoolFlag{
		Name:  "sync",
		Usage: "Enable/Disable sync",
	},
}

func listenForInterrupt() {
	intSignal = make(chan os.Signal, 1)
	signal.Notify(intSignal, os.Interrupt)
}

func readConfig(c *cli.Context) (config.Config, error) {
	if c.String("config") == "" {
		conf, err := config.ReadUserConfig()
		if err != nil {
			color.Yellow(err.Error())
			color.Yellow("Trying system configuration")
			return config.ReadSystemConfig()
		}

		return conf, err
	}

	return config.NewConfig(c.String("config"))
}

func startApp(c *cli.Context) error {
	conf, err := readConfig(c)
	if err != nil {
		color.Red("Failed to find a configuration file.")
		return err
	}

	db, err := database.NewDB(conf.Database)
	if err != nil {
		return err
	}

	sync := sync.NewService(db, conf.Sync)

	admin, err := admin.NewAdmin(db, conf.Admin.SocketPath)
	if err != nil {
		return err
	}
	admin.Start()

	defer admin.Stop(true)

	sync.Start()

	plugins := plugins.NewPlugins(conf.Plugins)

	listenForInterrupt()

	server := server.NewServer(db, &plugins, conf.Server)
	go func() {
		for sig := range intSignal {
			if sig == os.Interrupt || sig == os.Kill {
				err := server.Stop()
				if err != nil {
					color.Red(err.Error())
				}
			}
		}
	}()

	if err := server.Start(); err != nil {
		color.Red(err.Error())
	}

	return nil
}

func main() {
	app := cli.NewApp()

	app.Name = appName
	app.Usage = appUsage
	app.Flags = appFlags

	app.Action = startApp

	err := app.Run(os.Args)
	if err != nil {
		color.Red(err.Error())
	}
}
