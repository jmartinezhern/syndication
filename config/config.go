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

package config

import (
	"bufio"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// SystemConfigPath is Syndication's default path for system wide configuration.
	SystemConfigPath = "/etc/syndication/config.toml"

	// UserConfigRelativePath is the relative path to a user configuration.
	// This should be added to a full path to make a complete path to a configuration.
	// Internally when concatenate this with the full path $HOME/.config.
	UserConfigRelativePath = "syndication/config.toml"
)

type (
	// Server represents the complete configuration for Syndication's REST server component.
	Server struct {
		AuthSecret            string        `toml:"auth_secret"`
		AuthSecreteFilePath   string        `toml:"auth_secret_file_path"`
		EnableTLS             bool          `toml:"enable_tls"`
		EnableRequestLogs     bool          `toml:"enable_http_requests_log"`
		EnablePanicPrintStack bool          `toml:"enable_panic_print_stack"`
		Domain                string        `toml:"domain"`
		CertCacheDir          string        `toml:"cert_cache_dir"`
		MaxShutdownTime       int           `toml:"max_shutdown_time"`
		HTTPPort              int           `toml:"http_port"`
		ShutdownTimeout       time.Duration `toml:"shutdown_timeout"`
		APIKeyExpiration      time.Duration `toml:"api_key_expiration"`
		TLSPort               int           `toml:"tls_port"`
	}

	// Database represents the complete configuration for the database used by Syndication.
	Database struct {
		Type       string `toml:"-"`
		Enable     bool
		Connection string
	}

	// Sync represents configurations applicable to Syndication's sync component.
	Sync struct {
		SyncTime     time.Time     `toml:"time"`
		SyncInterval time.Duration `toml:"interval"`
	}

	// Admin represents configurations applicable to Syndication's admin component.
	Admin struct {
		SocketPath     string `toml:"socket_path"`
		MaxConnections int    `toml:"max_connections"`
	}

	// Config collects all configuration types
	Config struct {
		Database  `toml:"-"`
		Databases map[string]Database `toml:"database"`
		Server    Server
		Sync      Sync
		Admin     Admin
		path      string `toml:"-"`
	}
)

var (
	// DefaultDatabaseConfig represents the minimum configuration necessary for the database
	DefaultDatabaseConfig = Database{
		Type:       "sqlite3",
		Connection: "/var/syndication/syndication.db",
	}

	// DefaultServerConfig represents the minimum configuration necessary for the server component.
	DefaultServerConfig = Server{
		EnableRequestLogs:     false,
		EnablePanicPrintStack: true,
		AuthSecret:            "",
		AuthSecreteFilePath:   "",
		HTTPPort:              80,
	}

	// DefaultAdminConfig represents the minimum configuration necessary for the admin component.
	DefaultAdminConfig = Admin{
		SocketPath:     "/var/syndication/syndication.admin",
		MaxConnections: 5,
	}

	// DefaultSyncConfig represents the minimum configuration necessary for the sync component.
	DefaultSyncConfig = Sync{
		SyncInterval: time.Minute * 15,
	}

	// DefaultConfig collects all minimum default configurations.
	DefaultConfig = Config{
		Databases: map[string]Database{
			"sqlite": DefaultDatabaseConfig,
		},
		Admin:  DefaultAdminConfig,
		Server: DefaultServerConfig,
		Sync:   DefaultSyncConfig,
	}
)

type (
	// InvalidFieldValue represents an error caused by a query for an invalid field value.
	InvalidFieldValue struct {
		msg string
	}

	// FileSystemError signals that an file system error occurred during an operation.
	FileSystemError struct {
		msg string
	}

	// ParsingError signals that an error was encountered while parsing a configuration file.
	ParsingError struct {
		msg string
	}
)

func (c *Config) verifyConfig() error {
	if c.Server.AuthSecreteFilePath != "" {
		err := c.getSecretFromFile(c.Server.AuthSecreteFilePath)
		if err != nil {
			return err
		}
	} else if c.Server.AuthSecret == "" {
		return InvalidFieldValue{"Auth secret should not be empty"}
	}

	if len(c.Databases) == 0 {
		return InvalidFieldValue{"Configuration requires a database definition"}
	}

	for dbType, db := range c.Databases {
		if db.Enable {
			if c.Database != (Database{}) {
				log.Warn("Multiple database definitions are enabled. Using first found.")
				break
			}

			if dbType == "sqlite" {
				err := c.checkSQLiteConfig()
				if err != nil {
					return err
				}

				c.Database.Connection = c.Databases["sqlite"].Connection
				c.Database.Type = "sqlite3"
			} else if dbType == "mysql" {
				conn := c.Databases["mysql"].Connection
				if !strings.Contains(conn, "parseTime=True") {
					return InvalidFieldValue{"parseTime=True is required for a MySQL connection"}
				}

				c.Database.Connection = conn
				c.Database.Type = "mysql"
			} else if dbType == "postgres" {
				c.Database.Connection = c.Databases["postgres"].Connection
				c.Database.Type = "postgres"
			} else {
				log.Warn("Found unsupported database definition. Ignoring.")
			}
		}

	}

	if c.Database == (Database{}) {
		return InvalidFieldValue{"Database not defined or not enabled"}
	}

	return nil
}

func (c *Config) getSecretFromFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return InvalidFieldValue{"Invalid secrete file path"}
	}

	c.Server.AuthSecreteFilePath = absPath

	fi, err := os.Open(c.Server.AuthSecreteFilePath)
	if err != nil {
		return FileSystemError{"Could not read secrete file"}
	}

	r := bufio.NewReader(fi)
	buf := make([]byte, 512)
	if _, err := r.Read(buf); err != nil && err != io.EOF {
		return FileSystemError{"Could not read secrete file"}
	}

	c.Server.AuthSecret = string(buf)

	if err := fi.Close(); err != nil {
		return FileSystemError{"Could not close secrete file"}
	}

	return nil
}

func (c *Config) checkSQLiteConfig() error {
	path := c.Databases["sqlite"].Connection
	if path == "" {
		return InvalidFieldValue{"DB path cannot be empty"}
	}

	if !filepath.IsAbs(path) {
		return InvalidFieldValue{"DB path must be absolute"}
	}

	return nil
}

// Save the configuration
func (c *Config) Save() error {
	file, err := os.Create(c.path)
	if err != nil {
		return err
	}

	err = toml.NewEncoder(file).Encode(c)
	if err != nil {
		return err
	}

	return nil
}

// NewConfig creates new configuration from a file located at path.
func NewConfig(path string) (config Config, err error) {
	config.path = path

	_, err = os.Stat(path)
	if err != nil {
		return
	}

	_, err = toml.DecodeFile(path, &config)
	if err != nil {
		return
	}

	err = config.verifyConfig()
	if err != nil {
		config = Config{}
	}

	return
}

func (e InvalidFieldValue) Error() string {
	return e.msg
}

func (e FileSystemError) Error() string {
	return e.msg
}

func (e ParsingError) Error() string {
	return e.msg
}
