[![Build Status](https://travis-ci.org/varddum/syndication.svg?branch=master)](https://travis-ci.org/varddum/syndication)
[![codecov](https://codecov.io/gh/varddum/syndication/branch/master/graph/badge.svg)](https://codecov.io/gh/varddum/syndication)
[![Waffle.io - Columns and their card count](https://badge.waffle.io/varddum/syndication.svg?columns=all)](http://waffle.io/varddum/syndication)

# Syndication - A simple news aggregation server

## Features
* JSON REST API
* Unix socket based Administration API
* Let's Encrypt through Echo framework (not fully tested)
* Support for SQLite, MySQL and Postgres

## Building

```
$ export GOPATH=$(pwd)/syndication
$ git clone https://github.com/varddum/syndication $GOPATH/src/github.com/varddum/syndication
$ cd syndication/src/github.com/varddum/syndication
$ make
```
