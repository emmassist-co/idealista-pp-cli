.PHONY: build test lint install clean

build:
	go build -o bin/idealista-pp-cli ./cmd/idealista-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/idealista-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/idealista-pp-mcp ./cmd/idealista-pp-mcp

install-mcp:
	go install ./cmd/idealista-pp-mcp

build-all: build build-mcp
