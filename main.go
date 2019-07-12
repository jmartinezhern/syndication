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

	log "github.com/sirupsen/logrus"

	"github.com/jmartinezhern/syndication/cmd"
	"github.com/jmartinezhern/syndication/controller/rest"
	"github.com/jmartinezhern/syndication/repo/sql"
	"github.com/jmartinezhern/syndication/services"
	"github.com/jmartinezhern/syndication/sync"
)

func config() cmd.Config {
	if err := cmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}

	return cmd.EffectiveConfig
}

func main() {
	config := config()
	db := sql.NewDB(config.Database.Type, config.Database.Connection)

	server := rest.NewServer()

	usersRepo := sql.NewUsers(db)
	ctgsRepo := sql.NewCategories(db)
	entriesRepo := sql.NewEntries(db)
	feedsRepo := sql.NewFeeds(db)
	tagsRepo := sql.NewTags(db)

	authService := services.NewAuthService(config.AuthSecret, usersRepo)
	ctgsService := services.NewCategoriesService(ctgsRepo, entriesRepo)
	feedsService := services.NewFeedsService(feedsRepo, ctgsRepo, entriesRepo)
	entriesService := services.NewEntriesService(entriesRepo)
	tagsService := services.NewTagsService(tagsRepo, entriesRepo)
	usersService := services.NewUsersService(usersRepo)

	server.RegisterAuthService(authService, config.AuthSecret, config.AllowRegistrations)
	server.RegisterUsersService(usersService)
	server.RegisterCategoriesService(ctgsService)
	server.RegisterFeedsService(feedsService)
	server.RegisterEntriesService(entriesService)
	server.RegisterTagsController(tagsService)
	server.RegisterImporters(rest.Importers{"application/xml": services.NewOPMLImporter(ctgsRepo, feedsRepo)})
	server.RegisterExporters(rest.Exporters{"application/xml": services.NewOPMLExporter(ctgsRepo)})

	syncService := sync.NewService(config.Sync.Interval, feedsRepo, usersRepo, entriesRepo)
	syncService.Start()
	defer syncService.Stop()

	go func() {
		if err := server.Start(config.Host.Address, config.Host.Port); err != nil {
			log.Info("Shutting down...")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	if err := server.Stop(); err != nil {
		log.Warn(err)
	}
}
