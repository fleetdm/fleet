.PHONY: build

PATH := $(GOPATH)/bin:$(shell npm bin):$(PATH)

ifeq ($(OS), Windows_NT)
	GC_OFF = set GOGC=off &&
else
	GC_OFF = GOGC=off
endif

ifneq ($(OS), Windows_NT)
	ifeq ($(shell uname), Darwin)
		SHELL := /bin/bash
	endif
endif

ifeq ($(OS), Windows_NT)
	OUTPUT = build/kolide.exe
else
	OUTPUT = build/kolide
endif

VERSION = 0.0.0-development
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
USER = $(shell whoami)

ifeq ($(OS), Windows_NT)
	GOVERSION_CMD = "(go version).Split()[2]"
	GOVERSION = $(shell powershell $(GOVERSION_CMD))
	NOW	= $(shell powershell Get-Date -format s)
else
	GOVERSION = $(shell go version | awk '{print $$3}')
	NOW	= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
endif

DOCKER_IMAGE_NAME = kolide/kolide

ifndef CIRCLE_PR_NUMBER
	DOCKER_IMAGE_TAG = latest
else
	DOCKER_IMAGE_TAG = dev-${CIRCLE_PR_NUMBER}
endif

all: build

define HELP_TEXT

  Makefile commands

	make deps         - Install depedent programs and libraries
	make generate     - Generate and bundle required code
	make generate-dev - Generate and bundle required code in a watch loop
	make distclean    - Delete all build artifacts

	make build        - Build the code

	make test         - Run the full test suite
	make test-go      - Run the Go tests
	make test-js      - Run the JavaScript tests

	make lint         - Run all linters
	make lint-go      - Run the Go linters
	make lint-js      - Run the JavaScript linters
	make lint-scss    - Run the SCSS linters

	make run          - Run the Kolide server in dev mode

endef

help:
	$(info $(HELP_TEXT))

.prefix:
ifeq ($(OS), Windows_NT)
	if not exist build mkdir build
else
	mkdir -p build
endif

build: .prefix
	${GC_OFF} go build -i -o ${OUTPUT} -ldflags "\
	-X github.com/kolide/kolide-ose/server/version.version=${VERSION} \
	-X github.com/kolide/kolide-ose/server/version.branch=${BRANCH} \
	-X github.com/kolide/kolide-ose/server/version.revision=${REVISION} \
	-X github.com/kolide/kolide-ose/server/version.buildDate=${NOW} \
	-X github.com/kolide/kolide-ose/server/version.buildUser=${USER} \
	-X github.com/kolide/kolide-ose/server/version.goVersion=${GOVERSION}"

lint-js:
	eslint frontend --ext .js,.jsx
lint-ts:
	tslint frontend/**/*.tsx frontend/**/*.ts
lint-scss:
	sass-lint --verbose

lint-go:
	go vet $(shell glide nv)

lint: lint-go lint-js lint-scss lint-ts

test-go:
	go test -cover $(shell glide nv)

test-js:
	NODE_PATH=./frontend _mocha --compilers js:babel-core/register,tsx:typescript-require  \
		--recursive 'frontend/**/*.tests.js*' \
		--require ignore-styles \
		--require 'frontend/.test.setup.js' \
		--require 'frontend/test/loaderMock.js'

test: lint test-go test-js

generate: .prefix
	go-bindata -o=server/datastore/mysql/bindata.go -pkg=mysql db/
	webpack --progress --colors
	go-bindata -pkg=service \
		-o=server/service/bindata.go \
		frontend/templates/ assets/...


# we first generate the webpack bundle so that bindata knows to watch the
# output bundle file. then, generate debug bindata source file. finally, we
# run webpack in watch mode to continuously re-generate the bundle
generate-dev: .prefix
	webpack --progress --colors
	go-bindata -debug -pkg=service \
		-o=server/service/bindata.go \
		frontend/templates/ assets/...
	webpack --progress --colors --watch --notify

deps:
	npm install
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/Masterminds/glide
	glide install

distclean:
ifeq ($(OS), Windows_NT)
	if exist build rmdir /s/q build
	if exist vendor rmdir /s/q vendor
	if exist assets\bundle.js del assets\bundle.js
else
	rm -rf build vendor
	rm -f assets/bundle.js
endif

run:
	$(OUTPUT) serve --dev

docker-build-circle:
	@echo ">> building docker image"
	docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
	docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
