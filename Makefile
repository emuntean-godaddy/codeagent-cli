PACKAGES := $(shell go list ./...)
BINARY := codeagent
GOBIN := $(shell go env GOBIN)
GOPATH := $(shell go env GOPATH)
INSTALL_DIR := $(if $(GOBIN),$(GOBIN),$(GOPATH)/bin)

.PHONY: test
test:
	go test -race ./...

.PHONY: coverage
coverage:
	go test -race -coverprofile=coverage.out $(PACKAGES)

.PHONY: build
build:
	go build -o $(BINARY) .

.PHONY: install
install:
	go build -o $(INSTALL_DIR)/$(BINARY) .
