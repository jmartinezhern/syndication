[![Build Status](https://travis-ci.org/varddum/syndication.svg?branch=master)](https://travis-ci.org/varddum/syndication)
[![codecov](https://codecov.io/gh/varddum/syndication/branch/master/graph/badge.svg)](https://codecov.io/gh/varddum/syndication)
[![Waffle.io - Columns and their card count](https://badge.waffle.io/varddum/syndication.svg?columns=all)](http://waffle.io/varddum/syndication)

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
- [X] Tags

## Building

```
$ export GOPATH=$(pwd)/syndication
$ git clone https://github.com/varddum/syndication $GOPATH/src/github.com/varddum/syndication
$ cd syndication/src/github.com/varddum/syndication
$ make
```
