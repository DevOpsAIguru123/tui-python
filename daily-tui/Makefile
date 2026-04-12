APP_NAME  := daily-tui
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.version=$(VERSION)
PREFIX    ?= /usr/local

.PHONY: build install uninstall clean test

## build: compile the binary into ./bin/
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) .

## install: build and copy to $(PREFIX)/bin (default /usr/local/bin)
install: build
	install -d $(PREFIX)/bin
	install -m 755 bin/$(APP_NAME) $(PREFIX)/bin/$(APP_NAME)

## uninstall: remove the binary from $(PREFIX)/bin
uninstall:
	rm -f $(PREFIX)/bin/$(APP_NAME)

## clean: remove build artifacts
clean:
	rm -rf bin/ dist/

## test: run all tests
test:
	go test ./...
