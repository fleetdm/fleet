.PHONY: build

PATH := $(GOPATH)/bin:$(shell npm bin):$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)
DOCKER_IMAGE_NAME = kolide/kolide

ifneq ($(OS), Windows_NT)
	# If on macOS, set the shell to bash explicitly
	ifeq ($(shell uname), Darwin)
		SHELL := /bin/bash
	endif

	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = build/kolide

	# To populate version metadata, we use unix tools to get certain data
	GOVERSION = $(shell go version | awk '{print $$3}')
	NOW	= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
else
	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = build/kolide.exe

	# To populate version metadata, we use windows tools to get the certain data
	GOVERSION_CMD = "(go version).Split()[2]"
	GOVERSION = $(shell powershell $(GOVERSION_CMD))
	NOW	= $(shell powershell Get-Date -format s)
endif

ifndef CIRCLE_PR_NUMBER
	DOCKER_IMAGE_TAG = ${REVSHORT}
else
	DOCKER_IMAGE_TAG = dev-${CIRCLE_PR_NUMBER}-${REVSHORT}
endif

all: build

define HELP_TEXT

  Makefile commands

	make deps         - Install depedent programs and libraries
	make generate     - Generate and bundle required code
	make generate-dev - Generate and bundle required code in a watch loop
	make distclean    - Delete all build artifacts

	make build        - Build the code
	make package 	  - Build rpm and deb packages for linux

	make test         - Run the full test suite
	make test-go      - Run the Go tests
	make test-js      - Run the JavaScript tests

	make lint         - Run all linters
	make lint-go      - Run the Go linters
	make lint-js      - Run the JavaScript linters
	make lint-scss    - Run the SCSS linters
	make lint-ts      - Run the TypeScript linters


endef

help:
	$(info $(HELP_TEXT))

.prefix:
ifeq ($(OS), Windows_NT)
	if not exist build mkdir build
else
	mkdir -p build
endif

build: export GOGC = off
build: .prefix
	go build -i -o ${OUTPUT} -ldflags "\
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
	go test $(shell glide nv)

analyze-go:
	go test -race -cover $(shell glide nv)


test-js: export NODE_PATH = ./frontend
test-js:
	_mocha --compilers js:babel-core/register,tsx:typescript-require  \
		--recursive "frontend/**/*.tests.js*" \
		--require ignore-styles \
		--require "frontend/.test.setup.js" \
		--require "frontend/test/loaderMock.js"

test: lint test-go test-js

generate: .prefix
	NODE_ENV=production webpack --progress --colors
	go-bindata -pkg=service \
		-o=server/service/bindata.go \
		frontend/templates/ assets/...
	go-bindata -pkg=kolide -o=server/kolide/bindata.go server/mail/templates


# we first generate the webpack bundle so that bindata knows to watch the
# output bundle file. then, generate debug bindata source file. finally, we
# run webpack in watch mode to continuously re-generate the bundle
generate-dev: .prefix
	webpack --progress --colors
	go-bindata -debug -pkg=service \
		-o=server/service/bindata.go \
		frontend/templates/ assets/...
	go-bindata -pkg=kolide -o=server/kolide/bindata.go server/mail/templates
	webpack --progress --colors --watch --notify

deps:
	npm install
	go get github.com/jteeuwen/go-bindata/...
	go get github.com/Masterminds/glide
	go get github.com/groob/mockimpl
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


docker-build-circle:
	@echo ">> building docker image"
	docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
	docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"

demo-dump:
	mysqldump --extended-insert=FALSE --skip-dump-date \
		-u kolide -p \
		-h ${MYSQL_PORT_3306_TCP_ADDR} kolide \
		> ./tools/app/demo.sql

demo-restore:
	mysql --binary-mode -u kolide -p \
		-h ${MYSQL_PORT_3306_TCP_ADDR} kolide \
		< ./tools/app/demo.sql

package: export GOOS=linux
package: export CGO_ENABLED=0
package: build
	mkdir -p build/pkgroot/usr/bin
	cp build/kolide build/pkgroot/usr/bin
	docker run --rm -it -v ${PWD}/build/pkgroot:/pkgroot -v ${PWD}/build:/out -e KOLIDE_VERSION="${VERSION}" kolide/fpm

