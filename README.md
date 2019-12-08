![](https://github.com/jmartinezhern/syndication/workflows/Go/badge.svg)
[![codecov](https://codecov.io/gh/jmartinezhern/syndication/branch/master/graph/badge.svg)](https://codecov.io/gh/jmartinezhern/syndication)

# Syndication - A simple news aggregation server

## Features

- JSON REST API
- Let's Encrypt through Echo framework (experimental)
- Support for SQLite, MySQL and PostgreSQL

## Building

```bash
$ dep ensure
$ go build
```

## Usage

```bash
$ syndication --config synd.yaml
```

## Configuration

```yaml
# Authorization Secret. If not specified, syndication will
# generate one for you.
auth_secret: secret_cat

# Database configuration.
database:
  # Connection string for an SQL implementation. Examples:
  #   - mysql: user:password@/dbname?charset=utf8&parseTime=True&loc=Local
  #   - postgres: host=myhost port=myport user=synd dbname=synd password=mypassword
  connection: /var/lib/syndication.db

  # Connection Type. Can be one of the following:
  #   - mysql
  #   - postgres
  #   - sqlite3
  type: sqlite3

# Server configuration
host:
  address: localhost
  port: 8080

# Synchronization Configuration
sync:
  # How often to sync feeds
  interval: 15m0s
```
