.PHONY: all build test lint install clean run help

BINARY_NAME=ldin
BUILD_DIR=bin
VERSION=1.0.0
BUILD_DATE=$(shell date +%Y-%m-%d)
LDFLAGS=-ldflags "-X github.com/santusht/ldin/cmd.Version=$(VERSION) -X github.com/santusht/ldin/cmd.BuildDate=$(BUILD_DATE)"

all: build test

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	go test -v ./...

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin or $(GOPATH)/bin..."
	go install $(LDFLAGS) .

clean:
	rm -rf $(BUILD_DIR)
	go clean

help:
	@echo "Available targets:"
	@echo "  make build    - Compile binary to bin/ldin"
	@echo "  make test     - Run all unit test suites"
	@echo "  make install  - Install ldin into Go PATH"
	@echo "  make clean    - Remove build artifacts"
