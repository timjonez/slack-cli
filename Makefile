.PHONY: build test install clean

BINARY := slackcli
VERSION ?= 0.1.0
LDFLAGS := -X github.com/timjonez/slack-cli/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/slackcli

test:
	go test ./...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/slackcli

clean:
	rm -rf bin
