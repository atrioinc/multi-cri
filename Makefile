# Copyright 2018 AtrioInc
SHELL=/bin/bash -u

export GOPATH ?= $(shell echo $(CURDIR) | sed -e 's,/src/.*,,')
BASE    := ${PWD}
PACKAGE  := cri-babelfish

ABSOLUTE_PROJECT  = $(GOPATH)/src/cri-babelfish
PROJECT        := $(shell realpath --relative-to=$(CURDIR) $(ABSOLUTE_PROJECT))
FIND_PROJECT_DIRS  = find $(PROJECT) -mindepth 1 -maxdepth 1 -type d -not -name vendor -not -name '_vendor-*'
PROJECT_DIRS      := $(shell $(FIND_PROJECT_DIRS))
FIND_PROJECT_FILES = find $(PROJECT_DIRS) -name '*.go'
PROJECT_FILES     := $(shell $(FIND_PROJECT_FILES))

BIN     := $(shell realpath --relative-to=$(CURDIR) $(GOPATH)/bin/)
DEP ?= $(BIN)/dep



GOIMPORTS ?= $(BIN)/goimports$(BIN_ARCH)
GOIMPORTS_CMD = $(GOIMPORTS) -local cri-babelfish

.PHONY: all
all:    build

goimports: $(GOIMPORTS)
$(GOIMPORTS):
	@echo Building $(GOIMPORTS)...
	@go get -u golang.org/x/tools/cmd/goimports

dep: $(DEP)
$(DEP):
	@echo Building $(DEP)...
	@go get -u github.com/golang/dep/cmd/dep

tools: $(GOIMPORTS) $(DEP)

build: govendor
	@echo Building in bin/${PACKAGE}...
	@go build -o bin/${PACKAGE} main.go

build-static:
	@echo Building in bin/${PACKAGE}_static...
	@CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/${PACKAGE}_static main.go

clean:
	rm -rf bin
	rm -rf vendor
	rm -rf $(GOPATH)/bin
	rm -rf $(GOPATH)/pkg
	rm -rf $(GOPATH)/src/golang.org
	rm -rf $(GOPATH)/src/github.com

.PHONY: test
test:
	: Unit tests
	go test -run Unit ./...

.PHONY: integration
integration:
	: Integration tests
	go test -run Integration ./...

# install vendor packages required by Gopkg and original code
govendor: $(BASE)/vendor
$(BASE)/vendor: dep Gopkg.toml
	@echo Updating govendor dependencies
	@$(DEP) ensure -v

GOIMPORTS_PATCH = $(PROJECT)/.patch
GOIMPORTS_CMD = $(GOIMPORTS) -local rstor

.PHONY: imports
imports: $(GOIMPORTS)
	: Imports
	@$(GOIMPORTS_CMD) -d $(PROJECT_FILES) | tee $(GOIMPORTS_PATCH)
	@[[ ! -s $(GOIMPORTS_PATCH) ]]

.PHONY: fix
fix: $(GOIMPORTS)
	: Fix
	@$(GOIMPORTS_CMD) -d $(PROJECT_FILES)
	@$(GOIMPORTS_CMD) -w $(PROJECT_FILES)
