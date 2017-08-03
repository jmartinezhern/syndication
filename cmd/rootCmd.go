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
	"crypto/rand"
	"encoding/base64"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	generatedSecretLength = 128
	configFileName        = "config"
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

	// Sync configuration
	Sync struct {
		Interval    time.Duration
		DeleteAfter int `mapstructure:"delete_after"`
	}

	// Config represents a complete configuration
	Config struct {
		Sync       Sync
		EnableTLS  bool   `mapstructure:"enable_tls"`
		AuthSecret string `mapstructure:"auth_secret"`
		Database   Database
		Host       Host
		Admin      Admin
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

	if EffectiveConfig.AuthSecret == "" {
		secret := generateSecret()
		EffectiveConfig.AuthSecret = secret
		viper.Set("auth_secret", secret)

		if err := viper.WriteConfig(); err != nil {
			log.Error(err)
			os.Exit(1)
		}
	}

	return nil
}

func generateSecret() string {
	log.Info("No auth secret found. Generating new one...")

	b := make([]byte, generatedSecretLength)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(b)[0:generatedSecretLength]
}

func init() {
	rootCmd.Flags().StringVar(&cfgFile, "config", "", "config file")

	viper.SetDefault("sync.interval", time.Minute*15)
	viper.SetDefault("sync.delete_after", 30)
	viper.SetDefault("host.port", 8080)
	viper.SetDefault("host.address", "localhost")
	viper.SetDefault("database.type", "sqlite3")
	viper.SetDefault("database.connection", "/var/lib/syndication.db")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName(configFileName)
		viper.AddConfigPath("/etc/syndication")
		viper.AddConfigPath("$HOME/.config/syndication/config")
	}

	if err := viper.ReadInConfig(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
