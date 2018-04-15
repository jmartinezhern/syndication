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

package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type (
	// Plugin configuration
	Plugin struct {
		Name string
		Path string
	}

	// Admin configuration
	Admin struct {
		Enable     bool
		SocketPath string `mapstructure:"socket_path"`
	}

	// Host configuration
	Host struct {
		Address string
		Port    int
	}

	// Database configuration
	Database struct {
		Type       string
		Connection string
	}

	// Config represents a complete configuration
	Config struct {
		SyncInterval time.Duration `mapstructure:"sync_interval"`
		EnableTLS    bool          `mapstructure:"enable_tls"`
		Database     Database
		Host         Host
		Admin        Admin
	}
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use: "syndication",
}

// EffectiveConfig read by viper
var EffectiveConfig Config

// Execute the root command
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	initConfig()

	if err := viper.Unmarshal(&EffectiveConfig); err != nil {
		return err
	}

	if viper.ConfigFileUsed() == "" {
		err := readDatabaseTypeFromFlags()
		if err != nil {
			return err
		}

	}

	return nil
}

func readDatabaseTypeFromFlags() error {
	isSqlite, err := rootCmd.Flags().GetBool("sqlite")
	if err != nil {
		return err
	}

	if isSqlite {
		EffectiveConfig.Database.Type = "sqlite3"
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	rootCmd.Flags().Bool("admin", true, "enable admin")
	rootCmd.Flags().String("admin-socket", "/tmp/syndication.socket", "admin socket path")
	rootCmd.Flags().Bool("sqlite", true, "use sqlite db")
	rootCmd.Flags().String("db-connection", "/var/lib/syndication.db", "SQL DB specific connection")
	rootCmd.Flags().Duration("sync-interval", time.Minute*5, "sync interval")
	rootCmd.Flags().String("host", "localhost", "server host address")
	rootCmd.Flags().Int("port", 8080, "server host port")
	rootCmd.Flags().Bool("tls", false, "enable tls")

	err := viper.BindPFlag("host.address", rootCmd.Flags().Lookup("host"))

	if err == nil {
		err = viper.BindPFlag("host.port", rootCmd.Flags().Lookup("port"))
	}

	if err == nil {
		err = viper.BindPFlag("database.connection", rootCmd.Flags().Lookup("db-connection"))
	}

	if err == nil {
		err = viper.BindPFlag("sync_interval", rootCmd.Flags().Lookup("sync-interval"))
	}

	if err == nil {
		err = viper.BindPFlag("enable_tls", rootCmd.Flags().Lookup("tls"))
	}

	if err == nil {
		err = viper.BindPFlag("admin.enable", rootCmd.Flags().Lookup("admin"))
	}

	if err == nil {
		err = viper.BindPFlag("admin.socket_path", rootCmd.Flags().Lookup("admin-socket"))
	}

	if err != nil {
		panic(err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
	}
}
