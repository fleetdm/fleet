# Copyright (c) Facebook, Inc. and its affiliates.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

NAME = nvdtools
VERSION = tip

TOOLS = \
	cpe2cve \
	csv2cpe \
	fireeye2nvd \
	flexera2nvd \
	idefense2nvd \
	nvdsync \
	rpm2cpe \
	rustsec2nvd \
	snyk2nvd \
	vulndb

DOCS = \
	CODE_OF_CONDUCT.md \
	CONTRIBUTING.md \
	HOWTO.md \
	LICENSE \
	README.md

GO = go
GOOS = $(shell $(GO) env GOOS)
GOARCH = $(shell $(GO) env GOARCH)

TAR = tar
ZIP = zip
INSTALL = install

# Compile all tools.
all: $(TOOLS)

# Compile TOOLS to ./build/bin/$tool using GOOS and GOARCH.
$(TOOLS):
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -o ./build/bin/$@ ./cmd/$@

# Check/fetch all dependencies.
deps:
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) get -v -d ./...

# install installs tools and documentation.
# The install target is used by rpm and deb builders.
install:
	# tools
	$(INSTALL) -d $(DESTDIR)/usr/bin
	for tool in $(TOOLS); do $(INSTALL) -p -m 0755 ./build/bin/$$tool $(DESTDIR)/usr/bin/$$tool; done
	# docs
	$(INSTALL) -d $(DESTDIR)/usr/share/doc/nvdtools
	for doc in $(DOCS); do $(INSTALL) -p -m 0644 $$doc $(DESTDIR)/usr/share/doc/nvdtools/$$doc; done

DIST_NAME = $(NAME)-$(VERSION)
DIST_DIR = build/$(DIST_NAME)

# binary_dist creates a local binary distribution in DIST_DIR.
binary_dist: $(TOOLS)
	mkdir -p $(DIST_DIR)/doc
	cp $(DOCS) $(DIST_DIR)/doc
	mv build/bin $(DIST_DIR)/bin

# binary_tar creates tarball of binary distribution.
binary_tar: binary_dist
	mkdir -p build/tgz
	cd build && $(TAR) czf tgz/$(DIST_NAME)-$(GOOS)-$(GOARCH).tar.gz $(DIST_NAME)
	rm -rf $(DIST_DIR)

# binary_zip creates zip of binary distribution.
binary_zip: binary_dist
	mkdir -p build/zip
	cd build && $(ZIP) -r zip/$(DIST_NAME)-$(GOOS)-$(GOARCH).zip $(DIST_NAME)
	rm -rf $(DIST_DIR)

# binary_deb creates debian package.
#
# Requires GOPATH and dependencies available to compile nvdtools.
# Must set version to build: make binary_deb VERSION=1.0
binary_deb:
	VERSION=$(VERSION) dpkg-buildpackage -rfakeroot -uc -us
	mkdir -p build/deb
	mv ../$(NAME)*.deb build/deb/

# archive_tar creates tarball of the source code.
archive_tar:
	mkdir -p build/tgz
	$(TAR) czf build/tgz/$(DIST_NAME).tar.gz \
		--exclude=build \
		--exclude=release \
		--exclude=.git \
		--exclude=.travis.yml \
		--transform s/./$(DIST_NAME)/ \
		.

# binary_rpm creates rpm package.
#
# Requires GOPATH and dependencies available to compile nvdtools.
# Must set version to build: make binary_rpm VERSION=1.0
binary_rpm: archive_tar
	mkdir -p build/rpm/SOURCES
	mv build/tgz/$(DIST_NAME).tar.gz build/rpm/SOURCES/
	rpmbuild -ba \
		--define="_topdir $(PWD)/build/rpm" \
		--define="_version $(VERSION)" \
		nvdtools.spec

# release_tar creates tarball releases.
release_tar:
	mkdir -p release
	make deps binary_tar GOOS=darwin GOARCH=amd64
	make deps binary_tar GOOS=freebsd GOARCH=amd64
	make deps binary_tar GOOS=freebsd GOARCH=arm
	make deps binary_tar GOOS=linux GOARCH=amd64
	make deps binary_tar GOOS=linux GOARCH=arm64
	mv build/tgz/*.tar.gz release

# release_zip creates zip releases.
release_zip:
	mkdir -p release
	make deps binary_zip GOOS=windows GOARCH=386
	make deps binary_zip GOOS=windows GOARCH=amd64
	mv build/zip/*.zip release

# release_deb creates debian releases.
release_deb: binary_deb
	mkdir -p release
	mv build/deb/*.deb release

# release_rpm creates rpm releases.
release_rpm: binary_rpm
	mkdir -p release
	mv build/rpm/RPMS/*/*.rpm release

# release creates all release packages.
# Example: make distclean release VERSION=1.0
release: release_deb release_rpm release_tar release_zip

# Removes build related files.
clean:
	rm -rf build

distclean: clean
	rm -rf release

.PHONY: $(TOOLS)
