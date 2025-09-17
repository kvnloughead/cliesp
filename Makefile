## build: builds the binary for current platform
.PHONY: build
build:
	go build -o cliesp

## build-all: builds binaries for all platforms (mac, linux, windows)
.PHONY: build-all
build-all: build-mac build-linux build-windows

## build-mac: builds binary for macOS (Intel and Apple Silicon)
.PHONY: build-mac
build-mac:
	mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/cliesp-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o bin/cliesp-darwin-arm64

## build-linux: builds binary for Linux (amd64 and arm64)
.PHONY: build-linux
build-linux:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -o bin/cliesp-linux-amd64
	GOOS=linux GOARCH=arm64 go build -o bin/cliesp-linux-arm64

## build-windows: builds binary for Windows (amd64 and arm64)
.PHONY: build-windows
build-windows:
	mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/cliesp-windows-amd64.exe
	GOOS=windows GOARCH=arm64 go build -o bin/cliesp-windows-arm64.exe

## install: builds binary and moves to ~/.local/bin
.PHONY: install
install: build
	mv cliesp ~/.local/bin

# ============================================================
# HELPERS
# ============================================================

## help: print this help message
.PHONY: help
help:
	@echo "\nUsage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
	@echo "\nFlags: \n"
	@echo "  Command line flags are supported for run/api and run/air.\n  Specify them like this: "
	@echo "\n\t  make FLAGS=\"-x -y\" command"
	@echo "\n  For a list of implemented flags for the ./cmd/api application, \n  run 'make help/web'\n"
	@echo "\nEnvironmental Variables:\n"
	@echo "  Environmental variables are supported for run/api and run/air.\n  They can be exported to the environment, or stored in a .env file.\n"

.PHONY: confirm
confirm:
	@echo 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]
