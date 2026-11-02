.DEFAULT_GOAL := build
GOBIN ?= $(shell go env GOPATH)/bin

ifeq ($(filter-out /,$(abspath $(GOBIN))),)
$(error GOBIN is '$(GOBIN)'; it must name a real directory)
endif

.PHONY: build test clean push audition

AUDITION_SOURCE ?= .

build:
	go install ./...

test:
	go test ./... -count=1
	go vet ./...

# builds the working tree for the ubuntu 24.04 studio hosts; see docs/current/audition-build.md.
audition:
	bash tools/audition-build.sh "$(AUDITION_SOURCE)"

clean:
	go clean ./...
	rm -f "$(GOBIN)"/*
	rm -rf dist/

push: build
	push vendor "$(GOBIN)/sessionmixer" sessionmixer
