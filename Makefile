PROJECT=conair

BUILD_PATH := $(shell pwd)/
VERSION := $(shell cat VERSION)
PROJECT_PATH := $(BUILD_PATH)/src/github.com/giantswarm

BIN := $(PROJECT)

.PHONY: all clean get-deps fmt run-tests install

GOPATH := $(BUILD_PATH)

SOURCE=$(shell find . -name '*.go')

all: get-deps $(BIN)

clean:
	rm -rf $(BUILD_PATH)/src $(BUILD_PATH)/pkg $(BUILD_PATH)/bin $(BIN)

install: $(BIN)
	cp conair /usr/local/bin/

get-deps: src

src:
	mkdir -p $(PROJECT_PATH)
	cd "$(PROJECT_PATH)" && ln -s ../../.. $(PROJECT)

	#
	# Fetch private packages first (so `go get` skips them later)

	#
	# Fetch public dependencies via `go get`
	GOPATH=$(GOPATH) go get -d -v github.com/giantswarm/$(PROJECT)

	#
	# Build test packages (we only want those two, so we use `-d` in go get)
	#GOPATH=$(GOPATH) go get -d -v github.com/onsi/gomega

$(BIN): $(SOURCE) VERSION
	GOPATH=$(GOPATH) go build -ldflags "-X main.projectVersion $(VERSION)" -a -o $(BIN)

run-tests:
	GOPATH=$(GOPATH) go test ./...

fmt:
	gofmt -l -w .
