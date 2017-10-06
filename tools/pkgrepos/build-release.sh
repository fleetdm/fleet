#!/bin/bash

VERSION="$(git describe --tags --always --dirty)"
GPG_PATH="/Users/${USER}/.gnupg"

build_binaries() {
    cd $GOPATH/src/github.com/kolide/fleet
    make generate

    GOOS=darwin CGO_ENABLED=0 make build
    mkdir -p build/darwin
    mv build/fleet build/darwin/fleet_darwin_amd64

    GOOS=linux CGO_ENABLED=0 make build
    mkdir -p build/linux
    mv build/fleet build/linux/fleet_linux_amd64
}

zip_binaries() {
    cd build && \
        zip -r "fleet_${VERSION}.zip" darwin/ linux/ && \
        cp "fleet_${VERSION}.zip" fleet_latest.zip && \
        cd ..
}

build_linux_packages() {
	mkdir -p build/pkgroot/usr/bin
	cp build/linux/fleet_linux_amd64 build/pkgroot/usr/bin/fleet
	docker run --rm -it \
        -v ${PWD}/build/pkgroot:/pkgroot \
        -v "${GPG_PATH}:/root/.gnupg" \
        -v ${PWD}/build:/out -e KOLIDE_VERSION="${VERSION}" kolide/fpm
}

main() {
    build_binaries
    zip_binaries
    build_linux_packages
}

main
