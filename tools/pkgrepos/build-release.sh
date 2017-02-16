#!/bin/bash

VERSION="$(git describe --tags --always --dirty)"
GPG_PATH="/Users/${USER}/.gnupg"

build_binaries() {
    cd ../..
    GOOS=darwin CGO_ENABLED=0 make build 
    mkdir -p build/darwin
    mv build/kolide build/darwin/kolide_darwin_amd64

    GOOS=linux CGO_ENABLED=0 make build 
    mkdir -p build/linux
    mv build/kolide build/linux/kolide_linux_amd64
}

zip_binaries() {
    cd build && \
        zip -r "kolide_${VERSION}.zip" darwin/ linux/ && \
        cp "kolide_${VERSION}.zip" kolide_latest.zip && \
        cd ..
}

build_linux_packages() {
	mkdir -p build/pkgroot/usr/bin
	cp build/linux/kolide_linux_amd64 build/pkgroot/usr/bin/kolide
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
