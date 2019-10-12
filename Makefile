# Copyright (C) 2019 All Rights Reserved
# Author: Ivaylo Petrov ivaylo@ackl.io

.DEFAULT_GOAL := build

PROJECT_NAME := fitbit-data-exporter
BASE_PATH ?= github.com/ivajloip/

PKGS := $(shell go list ./...)

VERSION ?= $(shell git describe --always 2>/dev/null || echo "unknown")
GOOS ?= linux
GOARCH ?= amd64
GOMIPS = softfloat
FAST_BUILD ?= false
TARGET_DIR ?= build
MAIN_FILE_PATH ?= cmd/fde/main.go
IMAGE_REPO ?= ivajloip/$(PROJECT_NAME)
RUN_IMAGE_TAG ?= $(IMAGE_REPO):latest
RUN_IMAGE_VERSIONED_TAG ?= $(IMAGE_REPO):$(VERSION)
DEV_IMAGE_TAG ?= $(IMAGE_REPO):dev

.PHONY: build
build:
	@echo "Compiling source for $(GOOS) $(GOARCH)"
	@mkdir -p $(TARGET_DIR)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s -X main.version=$(VERSION)" \
		-o $(TARGET_DIR)/$(PROJECT_NAME)$(BINEXT) \
		$(MAIN_FILE_PATH)

.PHONY: build-static
build-static:
	@echo "Compiling source for $(GOOS) $(GOARCH) with static linking"
	@mkdir -p $(TARGET_DIR)
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
		-ldflags "-s -X main.version=$(VERSION)" \
		-a -installsuffix cgo \
		-o $(TARGET_DIR)/$(PROJECT_NAME)-static$(BINEXT) $(MAIN_FILE_PATH)

.PHONY: clean
clean:
	@echo "Cleaning up workspace"
	@rm -rf $(TARGET_DIR)
	@rm -rf dist/$(VERSION)

.PHONY: package
package: clean build
	@echo "Creating package for $(GOOS) $(GOARCH)"
	@mkdir -p dist/$(VERSION)
	@cp $(TARGET_DIR)/* dist/$(VERSION)
	@cd dist/$(VERSION)/ && tar -pczf ../$(PROJECT_NAME)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz .
	@rm -rf dist/$(VERSION)

# tests
.PHONY: code-lint
code-lint: ensure-test-tools
	@echo "Runningn static code checks"
	@echo "Running golint..."
	@golint -set_exit_status ./... || exit 1
	@echo "ok!"
	@echo "Running go vet..."
	@go vet $(PKGS)
	@echo "ok!"
	@echo "Running errcheck..."
	@errcheck $(PKGS)
	@echo "ok!"
	@echo "Running misspell..."
	@misspell $(PKGS)
	@echo "ok!"
	@echo "Running ineffassign..."
	@ineffassign . || exit 1
	@echo "ok!"
	@echo "Running gocyclo over 15..."
	@gocyclo -over 15 . || exit 1
	@echo "ok!"

.PHONY: ensure-test-tools
ensure-test-tools:
	@command -v golint >/dev/null 2>&1 || echo "Installing missing dependency golint" && go get golang.org/x/lint/golint
	@command -v errcheck >/dev/null 2>&1 || echo "Installing missing dependency errcheck" && go get github.com/kisielk/errcheck
	@command -v errcheck >/dev/null 2>&1 || echo "Installing missing dependency misspell" && go get -u github.com/client9/misspell/cmd/misspell
	@command -v errcheck >/dev/null 2>&1 || echo "Installing missing dependency ineffassign" && go get github.com/gordonklaus/ineffassign
	@command -v errcheck >/dev/null 2>&1 || echo "Installing missing dependency gocyclo" && go get github.com/fzipp/gocyclo

.PHONY: test
test: ensure-test-tools code-lint
	@echo "Running tests"
	@go test -cover -v $(PKGS)

# shortcuts for development

.PHONY: run
run: build
	./$(TARGET_DIR)/$(PROJECT_NAME)

.PHONY: run-docker-test
run-docker-test: build-dev-container
	@docker run --rm -ti --entrypoint /bin/sh $(BUILD_IMAGE_TAG)

.PHONY: build-dev-container
build-dev-container: $(DOCKER_TEMPORARY_EXPORT_FOLDER)
	@docker build --build-arg PROJECT_NAME=$(PROJECT_NAME) \
		--build-arg BASE_PATH=$(BASE_PATH) \
		--build-arg FAST_BUILD=$(FAST_BUILD) \
		--build-arg VERSION=$(VERSION) \
		--target dev \
		-f Dockerfile \
		-t $(DEV_IMAGE_TAG) \
		.

.PHONY: runnable-container
runnable-container: $(DOCKER_TEMPORARY_EXPORT_FOLDER)
	@docker build --build-arg PROJECT_NAME=$(PROJECT_NAME) \
		--build-arg BASE_PATH=$(BASE_PATH) \
		--build-arg FAST_BUILD=$(FAST_BUILD) \
		--build-arg VERSION=$(VERSION) \
		-f Dockerfile \
		-t $(RUN_IMAGE_TAG) \
		.

.PHONY: push-current-runnable-container
push-current-runnable-container: runnable-container
	@docker tag $(RUN_IMAGE_TAG) $(RUN_IMAGE_VERSIONED_TAG)
	@docker push $(RUN_IMAGE_VERSIONED_TAG)

.PHONY: push-latest-runnable-container
push-latest-runnable-container: runnable-container
	@docker push $(RUN_IMAGE_TAG)
