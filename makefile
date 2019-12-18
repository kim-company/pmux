# SPDX-FileCopyrightText: 2019 KIM KeepInMind GmbH
#
# SPDX-License-Identifier: MIT

VERSION          := $(shell git describe --tags --always --dirty="-dev")
COMMIT           := $(shell git rev-parse --short HEAD)
DATE             := $(shell date -u '+%Y-%m-%d-%H%M UTC')
VERSION_FLAGS    := -ldflags='-X "main.version=$(VERSION)" -X "main.commit=$(COMMIT)" -X "main.buildTime=$(DATE)"'

export GO111MODULE=on
MOCK=examples/mockcmd/main.go

.PHONY: all pmux mockcmd
all: pmux mockcmd

pmux: main.go
	go build -o bin/$@ $(VERSION_FLAGS) $^
mockcmd: $(MOCK)
	go build -o bin/$@ $^
test:
	go test ./...
format:
	go fmt ./...
clean:
	rm -rf bin
