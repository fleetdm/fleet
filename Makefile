.PHONY: build

export GO111MODULE=on

PATH := $(GOPATH)/bin:$(shell npm bin):$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)
DOCKER_IMAGE_NAME = fleetdm/fleet

ifneq ($(OS), Windows_NT)
	# If on macOS, set the shell to bash explicitly
	ifeq ($(shell uname), Darwin)
		SHELL := /bin/bash
	endif

	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = fleet

	# To populate version metadata, we use unix tools to get certain data
	GOVERSION = $(shell go version | awk '{print $$3}')
	NOW	= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
else
	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = fleet.exe

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

ifdef CIRCLE_TAG
	DOCKER_IMAGE_TAG = ${CIRCLE_TAG}
endif

ifndef MYSQL_PORT_3306_TCP_ADDR
	MYSQL_PORT_3306_TCP_ADDR = 127.0.0.1
endif

KIT_VERSION = "\
	-X github.com/kolide/kit/version.appName=${APP_NAME} \
	-X github.com/kolide/kit/version.version=${VERSION} \
	-X github.com/kolide/kit/version.branch=${BRANCH} \
	-X github.com/kolide/kit/version.revision=${REVISION} \
	-X github.com/kolide/kit/version.buildDate=${NOW} \
	-X github.com/kolide/kit/version.buildUser=${USER} \
	-X github.com/kolide/kit/version.goVersion=${GOVERSION}"

all: build

define HELP_TEXT

  Makefile commands

	make deps         - Install dependent programs and libraries
	make generate     - Generate and bundle required all code
	make generate-go  - Generate and bundle required go code
	make generate-js  - Generate and bundle required js code
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
	mkdir -p build/linux
	mkdir -p build/darwin
endif

.pre-build:
	$(eval GOGC = off)
	$(eval CGO_ENABLED = 0)

.pre-fleet:
	$(eval APP_NAME = fleet)

.pre-fleetctl:
	$(eval APP_NAME = fleetctl)

build: fleet fleetctl

fleet: .prefix .pre-build .pre-fleet
	CGO_ENABLED=0 go build -tags full -o build/${OUTPUT} -ldflags ${KIT_VERSION} ./cmd/fleet

fleetctl: .prefix .pre-build .pre-fleetctl
	CGO_ENABLED=0 go build -tags full -o build/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl

lint-js:
	yarn run eslint frontend --ext .js,.jsx

lint-ts:
	yarn run tslint frontend/**/*.tsx frontend/**/*.ts

lint-scss:
	yarn run sass-lint --verbose

lint-go:
	go vet ./...

lint: lint-go lint-js lint-scss lint-ts

test-go:
	go test -tags full ./...

analyze-go:
	go test -tags full -race -cover ./...

test-js:
	npm test

test: lint test-go test-js

generate: generate-js generate-go

generate-js: .prefix
	NODE_ENV=production webpack --progress --colors

generate-go: .prefix
	go-bindata -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates

# we first generate the webpack bundle so that bindata knows to watch the
# output bundle file. then, generate debug bindata source file. finally, we
# run webpack in watch mode to continuously re-generate the bundle
generate-dev: .prefix
	NODE_ENV=development webpack --progress --colors
	go-bindata -debug -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates
	NODE_ENV=development webpack --progress --colors --watch

deps: deps-js deps-go

deps-js:
	yarn

deps-go:
	GO111MODULE=off go get -u \
		github.com/kolide/go-bindata/... \
		github.com/golang/dep/cmd/dep \
		github.com/groob/mockimpl
	go mod download

distclean:
ifeq ($(OS), Windows_NT)
	if exist build rmdir /s/q build
	if exist vendor rmdir /s/q vendor
	if exist assets\bundle.js del assets\bundle.js
else
	rm -rf build vendor
	rm -f assets/bundle.js
endif

docker-build-release: xp-fleet xp-fleetctl
	docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
	docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" fleetdm/fleet:${VERSION}
	docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" fleetdm/fleet:latest

docker-push-release: docker-build-release
	docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
	docker push fleetdm/fleet:${VERSION}
	docker push fleetdm/fleet:latest

docker-build-circle:
	@echo ">> building docker image"
	CGO_ENABLED=0 GOOS=linux go build -o build/linux/${OUTPUT} -ldflags ${KIT_VERSION} ./cmd/fleet
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

.pre-binary-bundle:
	rm -rf build/binary-bundle
	mkdir -p build/binary-bundle/linux
	mkdir -p build/binary-bundle/darwin

xp-fleet: .pre-binary-bundle .pre-fleet generate
	CGO_ENABLED=0 GOOS=linux go build -tags full -o build/binary-bundle/linux/fleet -ldflags ${KIT_VERSION} ./cmd/fleet
	CGO_ENABLED=0 GOOS=darwin go build -tags full -o build/binary-bundle/darwin/fleet -ldflags ${KIT_VERSION} ./cmd/fleet
	CGO_ENABLED=0 GOOS=windows go build -tags full -o build/binary-bundle/windows/fleet.exe -ldflags ${KIT_VERSION} ./cmd/fleet

xp-fleetctl: .pre-binary-bundle .pre-fleetctl generate-go
	CGO_ENABLED=0 GOOS=linux go build -tags full -o build/binary-bundle/linux/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl
	CGO_ENABLED=0 GOOS=darwin go build -tags full -o build/binary-bundle/darwin/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl
	CGO_ENABLED=0 GOOS=windows go build -tags full -o build/binary-bundle/windows/fleetctl.exe -ldflags ${KIT_VERSION} ./cmd/fleetctl

binary-bundle: xp-fleet xp-fleetctl
	cd build/binary-bundle && zip -r fleet.zip darwin/ linux/ windows/
	cd build/binary-bundle && mkdir fleetctl-macos && cp darwin/fleetctl fleetctl-macos && tar -czf fleetctl-macos.tar.gz fleetctl-macos 
	cd build/binary-bundle && mkdir fleetctl-linux && cp linux/fleetctl fleetctl-linux && tar -czf fleetctl-linux.tar.gz fleetctl-linux 
	cd build/binary-bundle && mkdir fleetctl-windows && cp windows/fleetctl.exe fleetctl-windows && tar -czf fleetctl-windows.tar.gz fleetctl-windows 
	cd build/binary-bundle && shasum -a 256 fleet.zip fleetctl-macos.tar.gz fleetctl-windows.tar.gz fleetctl-linux.tar.gz

