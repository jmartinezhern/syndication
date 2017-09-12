# Syndication - An extensible news aggregation server

## Goals

**Simple.** Any web service can be a pain to install and configure. Syndication aims to make setup and maintenance as painless as possible.

**Extensible.** A Unix socket based administration API, a RESTful API and other features allow you to fine tune functionality according to your needs.

**Efficient.** Heavy testing, cyclic refactoring and constant static analysis are the some of the things used to ensure that Syndication is as fast and bug-free as it can be.

## Features
* JSON REST API
* Unix socket based Administration API
* Let's Encrypt through Echo framework
* Support for SQLite, MySQL and Postgres

## Planned Features
- [ ] Plugins
- [ ] Tags

## Building

```
$ export GOPATH=$(pwd)/syndication
$ git clone https://github.com/varddum/syndication $GOPATH/src/github.com/varddum/syndication
$ cd syndication/src/github.com/varddum/syndication
$ make
```
