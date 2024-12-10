#!/bin/bash

script_dir=$(dirname -- "$(readlink -f -- "$BASH_SOURCE")")
cd "$script_dir"

echo "Building fleetd deb package..."
fleetctl package --type=deb \
	--enable-scripts \
	--fleet-url=https://host.docker.internal:8080 \
	--enroll-secret=placeholder \
	--fleet-certificate=../osquery/fleet.crt \
    --debug
mv fleet-osquery_1*_amd64.deb fleet-osquery_amd64.deb

echo "Building fleetd rpm package..."
fleetctl package --type=rpm \
	--enable-scripts \
	--fleet-url=https://host.docker.internal:8080 \
	--enroll-secret=placeholder \
	--fleet-certificate=../osquery/fleet.crt \
    --debug
mv fleet-osquery-1*.x86_64.rpm fleet-osquery_amd64.rpm

echo "Building docker images..."
docker build -t fleetd-ubuntu-24.04 --platform=linux/amd64 -f ./ubuntu-24.04/Dockerfile .
docker build -t fleetd-fedora-41 --platform=linux/amd64 -f ./fedora-41/Dockerfile .
docker build -t fleetd-redhat-9.5 --platform=linux/amd64 -f ./redhat-9.5/Dockerfile .
docker build -t fleetd-centos-stream-10 --platform=linux/amd64 -f ./centos-stream-10/Dockerfile .
docker build -t fleetd-debian-12.8 --platform=linux/amd64 -f ./debian-12.8/Dockerfile .
docker build -t fleetd-amazonlinux-2023 --platform=linux/amd64 -f ./amazonlinux-2023/Dockerfile .
