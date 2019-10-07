VERSION          := $(shell git describe --tags --always --dirty="-dev")
COMMIT           := $(shell git rev-parse --short HEAD)
DATE             := $(shell date -u '+%Y-%m-%d-%H%M UTC')
VERSION_FLAGS    := -ldflags='-X "main.version=$(VERSION)" -X "main.commit=$(COMMIT)" -X "main.buildTime=$(DATE)"'

export GO111MODULE=on

.PHONY: all pmux
all: pmux
pmux:
	go build -o bin/pmux $(VERSION_FLAGS)
test:
	go test ./...
format:
	go fmt ./...
clean:
	rm -rf bin
