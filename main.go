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
	"os/user"

	"github.com/fatih/color"
	"github.com/urfave/cli"
	"github.com/varddum/syndication/admin"
	"github.com/varddum/syndication/config"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/server"
	"github.com/varddum/syndication/sync"
)

func findSystemConfig() (config.Config, error) {
	if _, err := os.Stat(config.SystemConfigPath); err != nil {
		return config.Config{}, err
	}

	conf, err := config.NewConfig(config.SystemConfigPath)
	if err != nil {
		return config.Config{}, err
	}

	return conf, nil
}

func findUserConfig() (config.Config, error) {
	currentUser, err := user.Current()
	if err != nil {
		return config.Config{}, err
	}

	configPath := currentUser.HomeDir + "/.config/" + config.UserConfigRelativePath
	if _, err = os.Stat(configPath); err != nil {
		return config.Config{}, err
	}

	conf, err := config.NewConfig(configPath)
	if err != nil {
		return config.Config{}, err
	}

	return conf, nil
}

func startApp(c *cli.Context) error {
	var conf config.Config
	var err error

	if c.String("config") == "" {
		conf, err = findUserConfig()
		if err != nil {
			color.Yellow(err.Error())
			color.Yellow("Trying system configuration")
			conf, err = findSystemConfig()
		}

		if err != nil {
			color.Yellow(err.Error())
			color.Red("Failed to find a configuration file.")
			return nil
		}
	} else {
		conf, err = config.NewConfig(c.String("config"))
		if err != nil {
			color.Red(err.Error())
			return nil
		}
	}

	db, err := database.NewDB(conf.Database.Type, conf.Database.Connection)
	if err != nil {
		return err
	}

	sync := sync.NewSync(db)

	admin, err := admin.NewAdmin(db, conf.Admin.SocketPath)
	if err != nil {
		return err
	}
	admin.Start()

	defer admin.Stop(true)

	sync.Start()

	server := server.NewServer(db, sync, conf.Server)
	if err = server.Start(); err != nil {
		color.Red(err.Error())
	}

	return err
}

func main() {
	app := cli.NewApp()

	app.Name = "syndication"
	app.Usage = "An flexible RSS server"

	app.Flags = []cli.Flag{
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

	app.Action = startApp

	err := app.Run(os.Args)
	if err != nil {
		color.Red(err.Error())
	}
}
