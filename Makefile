.PHONY: build clean clean-assets e2e-reset-db e2e-serve e2e-setup changelog db-reset db-backup db-restore

export GO111MODULE=on

PATH := $(shell npm bin):$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)
DOCKER_IMAGE_NAME = fleetdm/fleet

ifdef GO_TEST_EXTRA_FLAGS
GO_TEST_EXTRA_FLAGS_VAR := $(GO_TEST_EXTRA_FLAGS)
else
GO_TEST_EXTRA_FLAGS_VAR :=
endif

ifdef GO_BUILD_RACE_ENABLED
GO_BUILD_RACE_ENABLED_VAR := true
else
GO_BUILD_RACE_ENABLED_VAR := false
endif

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
	CGO_ENABLED=1 go build -race=${GO_BUILD_RACE_ENABLED_VAR} -tags full,fts5,netgo -o build/${OUTPUT} -ldflags ${KIT_VERSION} ./cmd/fleet

fleet-dev: GO_BUILD_RACE_ENABLED_VAR=true
fleet-dev: fleet

fleetctl: .prefix .pre-build .pre-fleetctl
	# Race requires cgo
	$(eval CGO_ENABLED := $(shell [[ "${GO_BUILD_RACE_ENABLED_VAR}" = "true" ]] && echo 1 || echo 0))
	CGO_ENABLED=${CGO_ENABLED} go build -race=${GO_BUILD_RACE_ENABLED_VAR} -o build/fleetctl -ldflags ${KIT_VERSION} ./cmd/fleetctl

fleetctl-dev: GO_BUILD_RACE_ENABLED_VAR=true
fleetctl-dev: fleetctl

lint-js:
	yarn lint

lint-go:
	golangci-lint run --skip-dirs ./node_modules --timeout 10m

lint: lint-go lint-js

dump-test-schema:
	go run ./tools/dbutils ./server/datastore/mysql/schema.sql

test-go: dump-test-schema generate-mock
	go test -tags full,fts5,netgo ${GO_TEST_EXTRA_FLAGS_VAR} -parallel 8 -coverprofile=coverage.txt -covermode=atomic ./cmd/... ./ee/... ./orbit/pkg/... ./orbit/cmd/orbit ./pkg/... ./server/... ./tools/...

analyze-go:
	go test -tags full,fts5,netgo -race -cover ./...

test-js:
	npm test

test: lint test-go test-js

generate: clean-assets generate-js generate-go

generate-ci:
	NODE_OPTIONS=--openssl-legacy-provider NODE_ENV=development yarn run webpack
	make generate-go

generate-js: clean-assets .prefix
	NODE_ENV=production yarn run webpack --progress

generate-go: .prefix
	go run github.com/kevinburke/go-bindata/go-bindata -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates

# we first generate the webpack bundle so that bindata knows to atch the
# output bundle file. then, generate debug bindata source file. finally, we
# run webpack in watch mode to continuously re-generate the bundle
generate-dev: .prefix
	NODE_ENV=development yarn run webpack --progress
	go run github.com/kevinburke/go-bindata/go-bindata -debug -pkg=bindata -tags full \
		-o=server/bindata/generated.go \
		frontend/templates/ assets/... server/mail/templates
	NODE_ENV=development yarn run webpack --progress --watch

generate-mock: .prefix
	go install github.com/fleetdm/mockimpl@ecbb3041eabfc9e046a3f2e414e32c28254b75b2
	go generate github.com/fleetdm/fleet/v4/server/mock github.com/fleetdm/fleet/v4/server/mock/mockresult github.com/fleetdm/fleet/v4/server/service/mock

generate-doc: .prefix
	go generate github.com/fleetdm/fleet/v4/server/fleet
	go generate github.com/fleetdm/fleet/v4/server/service/osquery_utils

deps: deps-js deps-go

deps-js:
	yarn

deps-go:
	go mod download

migration:
	go run github.com/fleetdm/goose/cmd/goose -dir server/datastore/mysql/migrations/tables create $(name)
	gofmt -w server/datastore/mysql/migrations/tables/*_$(name)*.go

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

fleetctl-docker: xp-fleetctl
	docker build -t fleetdm/fleetctl --platform=linux/amd64 -f tools/fleetctl-docker/Dockerfile .

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

# Build orbit/fleetd fleetd_tables extension
fleetd-tables-windows:
	GOOS=windows GOARCH=amd64 go build -o fleetd_tables_windows.exe ./orbit/cmd/fleetd_tables
fleetd-tables-linux:
	GOOS=linux GOARCH=amd64 go build -o fleetd_tables_linux.ext ./orbit/cmd/fleetd_tables
fleetd-tables-darwin:
	GOOS=darwin GOARCH=amd64 go build -o fleetd_tables_darwin.ext ./orbit/cmd/fleetd_tables
fleetd-tables-darwin_arm:
	GOOS=darwin GOARCH=arm64 go build -o fleetd_tables_darwin_arm.ext ./orbit/cmd/fleetd_tables
fleetd-tables-darwin-universal:
	$(MAKE) fleetd-tables-darwin fleetd-tables-darwin_arm
	lipo -create fleetd_tables_darwin.ext fleetd_tables_darwin_arm.ext -output fleetd_tables_darwin_universal.ext
fleetd-tables-all:
	$(MAKE) fleetd-tables-windows fleetd-tables-linux fleetd-tables-darwin-universal

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
	docker-compose exec -T mysql_test bash -c 'echo "drop database if exists e2e; create database e2e;" | MYSQL_PWD=toor mysql -uroot'
	./build/fleet prepare db --mysql_address=localhost:3307  --mysql_username=root --mysql_password=toor --mysql_database=e2e

e2e-setup:
	./build/fleetctl config set --context e2e --address https://localhost:8642 --tls-skip-verify true
	./build/fleetctl setup --context e2e --email=admin@example.com --password=password123# --org-name='Fleet Test' --name Admin
	./build/fleetctl user create --context e2e --email=maintainer@example.com --name maintainer --password=password123# --global-role=maintainer
	./build/fleetctl user create --context e2e --email=observer@example.com --name observer --password=password123# --global-role=observer
	./build/fleetctl user create --context e2e --email=sso_user@example.com --name "SSO user" --sso=true

# Setup e2e test environment and pre-populate database with software and vulnerabilities fixtures.
#
# Use in lieu of `e2e-setup` for tests that depend on these fixtures
e2e-setup-with-software:
	curl 'https://localhost:8642/api/v1/setup' \
		--data-raw '{"server_url":"https://localhost:8642","org_info":{"org_name":"Fleet Test"},"admin":{"admin":true,"email":"admin@example.com","name":"Admin","password":"password123#","password_confirmation":"password123#"}}' \
		--compressed \
		--insecure
	./tools/backup_db/restore_e2e_software_test.sh

e2e-serve-free: e2e-reset-db
	./build/fleet serve --mysql_address=localhost:3307 --mysql_username=root --mysql_password=toor --mysql_database=e2e --server_address=0.0.0.0:8642

e2e-serve-premium: e2e-reset-db
	./build/fleet serve  --dev_license --mysql_address=localhost:3307 --mysql_username=root --mysql_password=toor --mysql_database=e2e --server_address=0.0.0.0:8642

# Associate a host with a Fleet Desktop token.
#
# Usage:
# make e2e-set-desktop-token host_id=1 token=foo
e2e-set-desktop-token:
	docker-compose exec -T mysql_test bash -c 'echo "INSERT INTO e2e.host_device_auth (host_id, token) VALUES ($(host_id), \"$(token)\") ON DUPLICATE KEY UPDATE token=VALUES(token)" | MYSQL_PWD=toor mysql -uroot'

changelog:
	sh -c "find changes -type f | grep -v .keep | xargs -I {} sh -c 'grep \"\S\" {}; echo' > new-CHANGELOG.md"
	sh -c "cat new-CHANGELOG.md CHANGELOG.md > tmp-CHANGELOG.md && rm new-CHANGELOG.md && mv tmp-CHANGELOG.md CHANGELOG.md"
	sh -c "git rm changes/*"

changelog-orbit:
	sh -c "find orbit/changes -type file | grep -v .keep | xargs -I {} sh -c 'grep \"\S\" {}; echo' > new-CHANGELOG.md"
	sh -c "cat new-CHANGELOG.md orbit/CHANGELOG.md > tmp-CHANGELOG.md && rm new-CHANGELOG.md && mv tmp-CHANGELOG.md orbit/CHANGELOG.md"
	sh -c "git rm orbit/changes/*"

###
# Development DB commands
###

# Reset the development DB
db-reset:
	docker-compose exec -T mysql bash -c 'echo "drop database if exists fleet; create database fleet;" | MYSQL_PWD=toor mysql -uroot'
	./build/fleet prepare db --dev

# Back up the development DB to file
db-backup:
	./tools/backup_db/backup.sh

# Restore the development DB from file
db-restore:
	./tools/backup_db/restore.sh

# Generate osqueryd.app.tar.gz bundle from osquery.io.
#
# Usage:
# make osqueryd-app-tar-gz version=5.1.0 out-path=.
osqueryd-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target osqueryd-app-tar-gz is only supported on macOS"
	@exit 1
endif
	$(eval TMP_DIR := $(shell mktemp -d))
	curl -L https://github.com/osquery/osquery/releases/download/$(version)/osquery-$(version).pkg --output $(TMP_DIR)/osquery-$(version).pkg
	pkgutil --expand $(TMP_DIR)/osquery-$(version).pkg $(TMP_DIR)/osquery_pkg_expanded
	rm -rf $(TMP_DIR)/osquery_pkg_payload_expanded
	mkdir -p $(TMP_DIR)/osquery_pkg_payload_expanded
	tar xf $(TMP_DIR)/osquery_pkg_expanded/Payload --directory $(TMP_DIR)/osquery_pkg_payload_expanded
	$(TMP_DIR)/osquery_pkg_payload_expanded/opt/osquery/lib/osquery.app/Contents/MacOS/osqueryd --version
	tar czf $(out-path)/osqueryd.app.tar.gz -C $(TMP_DIR)/osquery_pkg_payload_expanded/opt/osquery/lib osquery.app
	rm -r $(TMP_DIR)

# Generate nudge.app.tar.gz bundle from nudge repo.
#
# Usage:
# make nudge-app-tar-gz version=1.1.10.81462 out-path=.
nudge-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target nudge-app-tar-gz is only supported on macOS"
	@exit 1
endif
	$(eval TMP_DIR := $(shell mktemp -d))
	curl -L https://github.com/macadmins/nudge/releases/download/v$(version)/Nudge-$(version).pkg --output $(TMP_DIR)/nudge-$(version).pkg
	pkgutil --expand $(TMP_DIR)/nudge-$(version).pkg $(TMP_DIR)/nudge_pkg_expanded
	mkdir -p $(TMP_DIR)/nudge_pkg_payload_expanded
	tar xvf $(TMP_DIR)/nudge_pkg_expanded/nudge-$(version).pkg/Payload --directory $(TMP_DIR)/nudge_pkg_payload_expanded
	$(TMP_DIR)/nudge_pkg_payload_expanded/Nudge.app/Contents/MacOS/Nudge --version
	tar czf $(out-path)/nudge.app.tar.gz -C $(TMP_DIR)/nudge_pkg_payload_expanded/ Nudge.app
	rm -r $(TMP_DIR)

# Generate swiftDialog.app.tar.gz bundle from the swiftDialog repo.
#
# Usage:
# make swift-dialog-app-tar-gz version=2.2.1 build=4591 out-path=.
swift-dialog-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target swift-dialog-app-tar-gz is only supported on macOS"
	@exit 1
endif
	# locking the version of swiftDialog to 2.2.1-4591 as newer versions
	# migth have layout issues.
ifneq ($(version), 2.2.1)
	@echo "Version is locked at 2.1.0, see comments in Makefile target for details"
	@exit 1
endif

ifneq ($(build), 4591)
	@echo "Build version is locked at 4591, see comments in Makefile target for details"
	@exit 1
endif
	$(eval TMP_DIR := $(shell mktemp -d))
	curl -L https://github.com/swiftDialog/swiftDialog/releases/download/v$(version)/dialog-$(version)-$(build).pkg --output $(TMP_DIR)/swiftDialog-$(version).pkg
	pkgutil --expand $(TMP_DIR)/swiftDialog-$(version).pkg $(TMP_DIR)/swiftDialog_pkg_expanded
	mkdir -p $(TMP_DIR)/swiftDialog_pkg_payload_expanded
	tar xvf $(TMP_DIR)/swiftDialog_pkg_expanded/tmp-package.pkg/Payload --directory $(TMP_DIR)/swiftDialog_pkg_payload_expanded
	$(TMP_DIR)/swiftDialog_pkg_payload_expanded/Library/Application\ Support/Dialog/Dialog.app/Contents/MacOS/Dialog --version
	tar czf $(out-path)/swiftDialog.app.tar.gz -C $(TMP_DIR)/swiftDialog_pkg_payload_expanded/Library/Application\ Support/Dialog/ Dialog.app
	rm -rf $(TMP_DIR)

# Build and generate desktop.app.tar.gz bundle.
#
# Usage:
# FLEET_DESKTOP_APPLE_AUTHORITY=foo FLEET_DESKTOP_VERSION=0.0.1 make desktop-app-tar-gz
#
# Output: desktop.app.tar.gz
desktop-app-tar-gz:
ifneq ($(shell uname), Darwin)
	@echo "Makefile target desktop-app-tar-gz is only supported on macOS"
	@exit 1
endif
	go run ./tools/desktop macos

FLEET_DESKTOP_VERSION ?= unknown

# Build desktop executable for Windows.
# This generates desktop executable for Windows that includes versioninfo binary properties
# These properties can be displayed when right-click on the binary in Windows Explorer.
# See: https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
# To sign this binary with a certificate, use signtool.exe or osslsigncode tool
#
# Usage:
# FLEET_DESKTOP_VERSION=0.0.1 make desktop-windows
#
# Output: fleet-desktop.exe
desktop-windows:
	go run ./orbit/tools/build/build-windows.go -version $(FLEET_DESKTOP_VERSION) -input ./orbit/cmd/desktop -output fleet-desktop.exe

# Build desktop executable for Linux.
#
# Usage:
# FLEET_DESKTOP_VERSION=0.0.1 make desktop-linux
#
# Output: desktop.tar.gz
desktop-linux:
	docker build -f Dockerfile-desktop-linux -t desktop-linux-builder .
	docker run --rm -v $(shell pwd):/output desktop-linux-builder /bin/bash -c "\
		mkdir /output/fleet-desktop && \
		go build -o /output/fleet-desktop/fleet-desktop -ldflags "-X=main.version=$(FLEET_DESKTOP_VERSION)" /usr/src/fleet/orbit/cmd/desktop && \
		cp /usr/lib/x86_64-linux-gnu/libayatana-appindicator3.so.1 \
		/usr/lib/x86_64-linux-gnu/libayatana-ido3-0.4.so.0 \
		/usr/lib/x86_64-linux-gnu/libayatana-indicator3.so.7 \
		/lib/x86_64-linux-gnu/libm.so.6 \
		/usr/lib/x86_64-linux-gnu/libcairo.so.2 \
		/usr/lib/x86_64-linux-gnu/libdbusmenu-gtk3.so.4 \
		/usr/lib/x86_64-linux-gnu/libdbusmenu-glib.so.4 \
		/output/fleet-desktop && cd /output && \
		tar czf desktop.tar.gz fleet-desktop && \
		rm -r fleet-desktop"

# Build orbit executable for Windows.
# This generates orbit executable for Windows that includes versioninfo binary properties
# These properties can be displayed when right-click on the binary in Windows Explorer.
# See: https://docs.microsoft.com/en-us/windows/win32/menurc/versioninfo-resource
# To sign this binary with a certificate, use signtool.exe or osslsigncode tool
#
# Usage:
# ORBIT_VERSION=0.0.1 make orbit-windows
#
# Output: orbit.exe
orbit-windows:
	go run ./orbit/tools/build/build-windows.go -version $(ORBIT_VERSION) -input ./orbit/cmd/orbit -output orbit.exe

# db-replica-setup setups one main and one read replica MySQL instance for dev/testing.
#	- Assumes the docker containers are already running (tools/mysql-replica-testing/docker-compose.yml)
# 	- MySQL instance listening on 3308 is the main instance.
# 	- MySQL instance listening on 3309 is the read instance.
#	- Sets a delay of 1s for replication.
db-replica-setup:
	$(eval MYSQL_REPLICATION_USER := replicator)
	$(eval MYSQL_REPLICATION_PASSWORD := rotacilper)
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "stop slave; reset slave all;"
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -AN -e "drop user if exists '$(MYSQL_REPLICATION_USER)'; create user '$(MYSQL_REPLICATION_USER)'@'%'; grant replication slave on *.* to '$(MYSQL_REPLICATION_USER)'@'%' identified by '$(MYSQL_REPLICATION_PASSWORD)'; flush privileges;"
	$(eval MAIN_POSITION := $(shell MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -e 'show master status \G' | grep Position | grep -o '[0-9]*'))
	$(eval MAIN_FILE := $(shell MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -e 'show master status \G' | grep File | sed -n -e 's/^.*: //p'))
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "change master to master_host='mysql_main',master_user='$(MYSQL_REPLICATION_USER)',master_password='$(MYSQL_REPLICATION_PASSWORD)',master_log_file='$(MAIN_FILE)',master_log_pos=$(MAIN_POSITION);"
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "change master to master_delay=1;"
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3309 -uroot -AN -e "start slave;"

# db-replica-reset resets the main MySQL instance.
db-replica-reset: fleet
	MYSQL_PWD=toor mysql --host 127.0.0.1 --port 3308 -uroot -e "drop database if exists fleet; create database fleet;"
	FLEET_MYSQL_ADDRESS=127.0.0.1:3308 ./build/fleet prepare db --dev

# db-replica-run runs fleet serve with one main and one read MySQL instance.
db-replica-run: fleet
	FLEET_MYSQL_ADDRESS=127.0.0.1:3308 FLEET_MYSQL_READ_REPLICA_ADDRESS=127.0.0.1:3309 FLEET_MYSQL_READ_REPLICA_USERNAME=fleet FLEET_MYSQL_READ_REPLICA_DATABASE=fleet FLEET_MYSQL_READ_REPLICA_PASSWORD=insecure ./build/fleet serve --dev --dev_license
