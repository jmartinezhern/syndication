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

	"github.com/varddum/syndication/admin"
	"github.com/varddum/syndication/cmd"
	"github.com/varddum/syndication/database"
	"github.com/varddum/syndication/server"
	"github.com/varddum/syndication/sync"

	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	config := cmd.EffectiveConfig

	if err := database.Init(
		config.Database.Type,
		config.Database.Connection,
	); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	sync := sync.NewService(config.Sync.Interval, config.Sync.DeleteAfter)
	sync.Start()
	defer sync.Stop()

	if config.Admin.Enable {
		admin, err := admin.NewService(config.Admin.SocketPath)
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
		admin.Start()

		defer admin.Stop()
	}

	server := server.NewServer(config.AuthSecret)

	go func() {
		if err := server.Start(config.Host.Address, config.Host.Port); err != nil {
			log.Info("Shutting down...")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	if err := server.Stop(); err != nil {
		log.Fatal(err)
	}
}
