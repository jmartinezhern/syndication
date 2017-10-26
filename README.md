[![Build Status](https://travis-ci.org/varddum/syndication.svg?branch=master)](https://travis-ci.org/varddum/syndication)
[![codecov](https://codecov.io/gh/varddum/syndication/branch/master/graph/badge.svg)](https://codecov.io/gh/varddum/syndication)
[![Waffle.io - Columns and their card count](https://badge.waffle.io/varddum/syndication.svg?columns=all)](http://waffle.io/varddum/syndication)

# Syndication - An extensible news aggregation server

## Features
* JSON REST API
* Unix socket based Administration API
* Let's Encrypt through Echo framework (not fully tested)
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

## Contributing

I built this for my self but I would like to make this useful for everyone. The best way to do this is through feedback. Please feel free to make pull requests and report issues.
