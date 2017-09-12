SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -name '*.go' -not -path vendor)

TARGET=syndication

VERSION=0.0.0

LDFLAGS=-ldflags "-X github.com/varddum/syndication/core.Version=${VERSION}"

.PHONY: install
.PHONY: clean
.PHONY: depends

build: $(TARGET)

$(TARGET): depends
	go build ${LDFLAGS} -o ${TARGET} main.go

depends: $(SOURCES)
	glide install .

install:
	go install ${LDFLAGS} ./...

clean:
	rm $(TARGET)
