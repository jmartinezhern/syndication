#!/usr/bin/env bash

set -e

test -z "$(gometalinter -j 4 --disable-all \
  --enable=gofmt \
  --enable=golint \
  --enable=vet \
  --vendor \
  --deadline=10m ./... 2>&1 | egrep -v 'testdata/' | tee /dev/stderr)"

echo "" > coverage.txt

for d in $(go list ./... | grep -v vendor); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
  done
