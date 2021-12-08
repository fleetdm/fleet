.PHONY: build clean clean-assets e2e-reset-db e2e-serve e2e-setup changelog db-reset db-backup db-restore

export GO111MODULE=on

PATH := $(shell npm bin):$(PATH)
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
	NOW	= $(shell date +"%Y-%m-%d")
else
	# The output binary name is different on Windows, so we're explicit here
	OUTPUT = fleet.exe

	# To populate version metadata, we use windows tools to get the certain data
	GOVERSION_CMD = "(go version).Split()[2]"
	GOVERSION = $(shell powershell $(GOVERSION_CMD))
	NOW	= $(shell powershell Get-Date -format "yyy-MM-dd")
endif

ifndef CIRCLE_PR_NUMBER
	DOCKER_IMAGE_TAG = ${REVSHORT}
else
	DOCKER_IMAGE_TAG = dev-${CIRCLE_PR_NUMBER}-${REVSHORT}
endif

ifdef CIRCLE_TAG
	DOCKER_IMAGE_TAG = ${CIRCLE_TAG}
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

    make clean        - Clean all build artifacts
	make clean-assets - Clean assets only

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
	mkdir -p build/linux
	mkdir -p build/darwin

.pre-build:
	$(eval GOGC = off)
	$(eval CGO_ENABLED = 0)

.pre-fleet:
	$(eval APP_NAME = fleet)

.pre-fleetctl:
	$(eval APP_NAME = fleetctl)

build: fleet fleetctl

fleet: .prefix .pre-build .pre-fleet
	CGO_ENABLED=1 go build -tags full,fts5,netgo -o build/${OUTPUT} -ldflags ${KIT_VERSION} ./cmd/fleet

fleetctl: .prefix .pre-build .pre-fleetctl
	CGO_ENABLED=0 go build -o build/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl

lint-js:
	yarn lint

lint-go:
	golangci-lint run --skip-dirs ./node_modules

lint: lint-go lint-js

dump-test-schema:
	go run ./tools/dbutils ./server/datastore/mysql/schema.sql

test-go: dump-test-schema generate-mock
	go test -tags full,fts5,netgo -parallel 8 -coverprofile=coverage.txt -covermode=atomic ./cmd/... ./ee/... ./orbit/... ./pkg/... ./server/... ./tools/...

analyze-go:
	go test -tags full,fts5,netgo -race -cover ./...

test-js:
	npm test

test: lint test-go test-js

generate: clean-assets generate-js generate-go

generate-ci:
	NODE_ENV=development webpack
	make generate-go

generate-js: clean-assets .prefix
	NODE_ENV=production webpack --progress --colors

generate-go: .prefix
	go run github.com/kevinburke/go-bindata/go-bindata -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates

# we first generate the webpack bundle so that bindata knows to atch the
# output bundle file. then, generate debug bindata source file. finally, we
# run webpack in watch mode to continuously re-generate the bundle
generate-dev: .prefix
	NODE_ENV=development webpack --progress --colors
	go run github.com/kevinburke/go-bindata/go-bindata -debug -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates
	NODE_ENV=development webpack --progress --colors --watch

generate-mock: .prefix
	go install github.com/groob/mockimpl@latest
	go generate github.com/fleetdm/fleet/v4/server/mock

deps: deps-js deps-go

deps-js:
	yarn

deps-go:
	go mod download

migration:
	go run github.com/fleetdm/goose/cmd/goose -dir server/datastore/mysql/migrations/tables create $(name)

clean: clean-assets
	rm -rf build vendor
	rm -f assets/bundle.js

clean-assets:
	git clean -fx assets

docker-build-release: xp-fleet xp-fleetctl
	docker build -t "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" .
	docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" fleetdm/fleet:${VERSION}
	docker tag "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}" fleetdm/fleet:latest

docker-push-release: docker-build-release
	docker push "${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
	docker push fleetdm/fleet:${VERSION}
	docker push fleetdm/fleet:latest

.pre-binary-bundle:
	rm -rf build/binary-bundle
	mkdir -p build/binary-bundle/linux
	mkdir -p build/binary-bundle/darwin

xp-fleet: .pre-binary-bundle .pre-fleet generate
	CGO_ENABLED=1 GOOS=linux go build -tags full,fts5,netgo -trimpath -o build/binary-bundle/linux/fleet -ldflags ${KIT_VERSION} ./cmd/fleet
	CGO_ENABLED=1 GOOS=darwin go build -tags full,fts5,netgo -trimpath -o build/binary-bundle/darwin/fleet -ldflags ${KIT_VERSION} ./cmd/fleet
	CGO_ENABLED=1 GOOS=windows go build -tags full,fts5,netgo -trimpath -o build/binary-bundle/windows/fleet.exe -ldflags ${KIT_VERSION} ./cmd/fleet

xp-fleetctl: .pre-binary-bundle .pre-fleetctl generate-go
	CGO_ENABLED=0 GOOS=linux go build -trimpath -o build/binary-bundle/linux/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl
	CGO_ENABLED=0 GOOS=darwin go build -trimpath -o build/binary-bundle/darwin/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl
	CGO_ENABLED=0 GOOS=windows go build -trimpath -o build/binary-bundle/windows/fleetctl.exe -ldflags ${KIT_VERSION} ./cmd/fleetctl

binary-bundle: xp-fleet xp-fleetctl
	cd build/binary-bundle && zip -r fleet.zip darwin/ linux/ windows/
	cd build/binary-bundle && mkdir fleetctl-macos && cp darwin/fleetctl fleetctl-macos && tar -czf fleetctl-macos.tar.gz fleetctl-macos
	cd build/binary-bundle && mkdir fleetctl-linux && cp linux/fleetctl fleetctl-linux && tar -czf fleetctl-linux.tar.gz fleetctl-linux
	cd build/binary-bundle && mkdir fleetctl-windows && cp windows/fleetctl.exe fleetctl-windows && tar -czf fleetctl-windows.tar.gz fleetctl-windows
	cd build/binary-bundle && cp windows/fleetctl.exe . && zip fleetctl.exe.zip fleetctl.exe
	cd build/binary-bundle && shasum -a 256 fleet.zip fleetctl.exe.zip fleetctl-macos.tar.gz fleetctl-windows.tar.gz fleetctl-linux.tar.gz


.pre-binary-arch:
ifndef GOOS
	@echo "GOOS is Empty. Try use to see valid GOOS/GOARCH platform: go tool dist list. Ex.: make binary-arch GOOS=linux GOARCH=arm64"
	@exit 1;
endif
ifndef GOARCH
	@echo "GOARCH is Empty. Try use to see valid GOOS/GOARCH platform: go tool dist list. Ex.: make binary-arch GOOS=linux GOARCH=arm64"
	@exit 1;
endif


binary-arch: .pre-binary-arch .pre-binary-bundle .pre-fleet
	mkdir -p build/binary-bundle/${GOARCH}-${GOOS}
	CGO_ENABLED=1 GOARCH=${GOARCH} GOOS=${GOOS} go build -tags full,fts5,netgo -o build/binary-bundle/${GOARCH}-${GOOS}/fleet -ldflags ${KIT_VERSION} ./cmd/fleet
	CGO_ENABLED=0 GOARCH=${GOARCH} GOOS=${GOOS} go build -tags full,fts5,netgo -o build/binary-bundle/${GOARCH}-${GOOS}/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl
	cd build/binary-bundle/${GOARCH}-${GOOS} && tar -czf fleetctl-${GOARCH}-${GOOS}.tar.gz fleetctl fleet


# Drop, create, and migrate the e2e test database
e2e-reset-db:
	docker-compose exec -T mysql_test bash -c 'echo "drop database if exists e2e; create database e2e;" | mysql -uroot -ptoor'
	./build/fleet prepare db --mysql_address=localhost:3307  --mysql_username=root --mysql_password=toor --mysql_database=e2e

e2e-setup:
	./build/fleetctl config set --context e2e --address https://localhost:8642 --tls-skip-verify true
	./build/fleetctl setup --context e2e --email=admin@example.com --password=user123# --org-name='Fleet Test' --name Admin
	./build/fleetctl user create --context e2e --email=maintainer@example.com --name maintainer --password=user123# --global-role=maintainer
	./build/fleetctl user create --context e2e --email=observer@example.com --name observer --password=user123# --global-role=observer
	./build/fleetctl user create --context e2e --email=sso_user@example.com --name "SSO user" --sso=true

e2e-serve-free:
	./build/fleet serve --mysql_address=localhost:3307 --mysql_username=root --mysql_password=toor --mysql_database=e2e --server_address=0.0.0.0:8642

e2e-serve-premium:
	./build/fleet serve  --dev_license --mysql_address=localhost:3307 --mysql_username=root --mysql_password=toor --mysql_database=e2e --server_address=0.0.0.0:8642

changelog:
	sh -c "find changes -type file | grep -v .keep | xargs -I {} sh -c 'grep \"\S\" {}; echo' > new-CHANGELOG.md"
	sh -c "cat new-CHANGELOG.md CHANGELOG.md > tmp-CHANGELOG.md && rm new-CHANGELOG.md && mv tmp-CHANGELOG.md CHANGELOG.md"
	sh -c "git rm changes/*"

###
# Development DB commands
###

# Reset the development DB
db-reset:
	docker-compose exec -T mysql bash -c 'echo "drop database if exists fleet; create database fleet;" | mysql -uroot -ptoor'
	./build/fleet prepare db --dev

# Back up the development DB to file
db-backup:
	./tools/backup_db/backup.sh

# Restore the development DB from file
db-restore:
	./tools/backup_db/restore.sh
