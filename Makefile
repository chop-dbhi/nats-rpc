PROG_NAME := nats-rpc

GIT_VERSION := $(shell git log -1 --pretty=format:"%h (%ci)")

GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

deps:
	go get -u github.com/golang/dep/...
	dep ensure

build:
	mkdir -p dist

	go build \
		-ldflags "-X \"main.buildVersion=$(GIT_VERSION)\" -extldflags -static" \
		-o ./dist/$(GOOS)-$(GOARCH)/$(PROG_NAME) .
