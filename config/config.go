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

// Duration represents a duration string such as "3m4s" as a time.Duration
type Duration struct {
	time.Duration
}

// UnmarshalText decodes a toml duration string such as "5m1s"
func (d *Duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

const (
	// SystemConfigPath is Syndication's default path for system wide configuration.
	SystemConfigPath = "/etc/syndication/config.toml"

	// UserConfigRelativePath is the relative path to a user configuration.
	// This should be concatenaded with the running user's home directory.
	UserConfigRelativePath = "syndication/config.toml"
)

type (
	// Server represents the complete configuration for Syndication's REST server component.
	Server struct {
		AuthSecret            string   `toml:"auth_secret"`
		AuthSecreteFilePath   string   `toml:"auth_secret_file_path"`
		EnableTLS             bool     `toml:"enable_tls"`
		EnableRequestLogs     bool     `toml:"enable_http_requests_log"`
		EnablePanicPrintStack bool     `toml:"enable_panic_print_stack"`
		Domain                string   `toml:"domain"`
		CertCacheDir          string   `toml:"cert_cache_dir"`
		MaxShutdownTime       int      `toml:"max_shutdown_time"`
		HTTPPort              int      `toml:"http_port"`
		ShutdownTimeout       Duration `toml:"shutdown_timeout"`
		TLSPort               int      `toml:"tls_port"`
	}

	// Database represents the complete configuration for the database used by Syndication.
	Database struct {
		Type             string `toml:"-"`
		Enable           bool
		Connection       string
		APIKeyExpiration Duration `toml:"api_key_expiration"`
	}

	// Sync represents configurations applicable to Syndication's sync component.
	Sync struct {
		SyncTime     time.Time `toml:"time"`
		SyncInterval Duration  `toml:"interval"`
	}

	// Admin represents configurations applicable to Syndication's admin component.
	Admin struct {
		SocketPath     string `toml:"socket_path"`
		MaxConnections int    `toml:"max_connections"`
	}

	// Config collects all configuration types
	Config struct {
		Plugins   []string
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
		Type:             "sqlite3",
		Connection:       "/var/syndication/syndication.db",
		APIKeyExpiration: Duration{time.Hour * 72},
	}

	// DefaultServerConfig represents the minimum configuration necessary for the server component.
	DefaultServerConfig = Server{
		EnableRequestLogs:     false,
		EnablePanicPrintStack: true,
		AuthSecret:            "",
		AuthSecreteFilePath:   "",
		HTTPPort:              80,
		TLSPort:               443,
	}

	// DefaultAdminConfig represents the minimum configuration necessary for the admin component.
	DefaultAdminConfig = Admin{
		SocketPath:     "/var/syndication/syndication.admin",
		MaxConnections: 5,
	}

	// DefaultSyncConfig represents the minimum configuration necessary for the sync component.
	DefaultSyncConfig = Sync{
		SyncInterval: Duration{time.Minute * 15},
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
	err := c.parseDatabase()
	if err != nil {
		return err
	}

	err = c.parseAdmin()
	if err != nil {
		return err
	}

	err = c.parseSync()
	if err != nil {
		return err
	}

	err = c.parsePlugins()
	if err != nil {
		return err
	}

	return c.parseServer()
}

func (c *Config) getSecretFromFile(path string) error {
	if !filepath.IsAbs(path) {
		return InvalidFieldValue{"Invalid secrete file path"}
	}

	c.Server.AuthSecreteFilePath = path

	fi, err := os.Open(c.Server.AuthSecreteFilePath)
	if err != nil {
		return err
	}

	r := bufio.NewReader(fi)
	buf := make([]byte, 512)
	if _, err := r.Read(buf); err != nil && err != io.EOF {
		return err
	}

	c.Server.AuthSecret = string(buf)

	return fi.Close()
}

func (c *Config) parsePlugins() error {
	var validPlugins []string
	for _, plugin := range c.Plugins {
		_, err := os.Stat(plugin)
		if err != nil {
			log.Warn(err, ". Skipping")
			continue
		}

		validPlugins = append(validPlugins, plugin)
	}

	c.Plugins = validPlugins

	return nil
}

func (c *Config) parseServer() error {
	if c.Server.AuthSecreteFilePath != "" {
		err := c.getSecretFromFile(c.Server.AuthSecreteFilePath)
		if err != nil {
			return err
		}
	} else if c.Server.AuthSecret == "" {
		return InvalidFieldValue{"Auth secret should not be empty"}
	}

	if c.Server.HTTPPort == 0 {
		c.Server.HTTPPort = DefaultServerConfig.HTTPPort
	}

	if c.Server.TLSPort == 0 {
		c.Server.TLSPort = DefaultServerConfig.TLSPort
	}

	return nil
}

func (c *Config) parseAdmin() error {
	if c.Admin.SocketPath != "" {
		if !filepath.IsAbs(c.Admin.SocketPath) {
			return InvalidFieldValue{"Admin socket path must be absolute"}
		}
	} else {
		c.Admin.SocketPath = DefaultAdminConfig.SocketPath
	}

	if c.Admin.MaxConnections == 0 {
		c.Admin.MaxConnections = DefaultAdminConfig.MaxConnections
	}

	return nil
}

func (c *Config) parseSync() error {
	if c.Sync.SyncInterval.Duration == 0 {
		c.Sync.SyncInterval = DefaultSyncConfig.SyncInterval
	} else if c.Sync.SyncInterval.Minutes() < time.Duration(time.Minute*5).Minutes() {
		return InvalidFieldValue{"Sync interval should be 5 minutes or greater"}
	}

	return nil
}

func (c *Config) parseDatabase() error {
	c.Database = Database{}

	for dbType, db := range c.Databases {
		if !db.Enable {
			continue
		} else if c.Database != (Database{}) {
			log.Warn("Multiple database definitions are enabled. Using first found.")
			break
		}

		var err error
		switch dbType {
		case "sqlite":
			err = c.parseSQLiteDB()
		case "mysql":
			err = c.parseMySQLDB()
		case "postgres":
			err = c.parsePostgresDB()
		default:
			log.Warn("Found unsupported database definition. Ignoring.")
		}

		if err != nil {
			log.Error(err)
		}
	}

	if c.Database == (Database{}) {
		return InvalidFieldValue{"Database not defined or not enabled"}
	}

	if c.Database.APIKeyExpiration.Duration == 0 {
		c.Database.APIKeyExpiration = DefaultDatabaseConfig.APIKeyExpiration
	}

	return nil
}

func (c *Config) parseSQLiteDB() error {
	path := c.Databases["sqlite"].Connection
	if path == "" {
		return InvalidFieldValue{"DB path cannot be empty"}
	}

	if !filepath.IsAbs(path) {
		return InvalidFieldValue{"DB path must be absolute"}
	}

	c.Database.Connection = c.Databases["sqlite"].Connection
	c.Database.Type = "sqlite3"

	return nil
}

func (c *Config) parseMySQLDB() error {
	conn := c.Databases["mysql"].Connection
	if !strings.Contains(conn, "parseTime=True") {
		return InvalidFieldValue{"parseTime=True is required for a MySQL connection"}
	}

	c.Database.Connection = conn
	c.Database.Type = "mysql"

	return nil
}

func (c *Config) parsePostgresDB() error {
	c.Database.Connection = c.Databases["postgres"].Connection
	c.Database.Type = "postgres"

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
		log.Error(err)
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
