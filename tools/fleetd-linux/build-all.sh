#!/bin/bash

set -euo pipefail

script_dir=$(dirname -- "$(readlink -f -- "$BASH_SOURCE")")
cd "$script_dir"

echo "Building fleetd deb (amd64) package..."
fleetctl package --type=deb \
	--enable-scripts \
	--fleet-url=https://host.docker.internal:8080 \
	--enroll-secret=placeholder \
	--fleet-certificate=../osquery/fleet.crt \
	--disable-open-folder \
	--outfile=fleet-osquery_amd64.deb \
    ${UPDATE_URL:+--update-url=$UPDATE_URL} \
    ${ORBIT_CHANNEL:+--orbit-channel=$ORBIT_CHANNEL} \
    ${DESKTOP_CHANNEL:+--desktop-channel=$DESKTOP_CHANNEL} \
    ${OSQUERYD_CHANNEL:+--osqueryd-channel=$OSQUERYD_CHANNEL} \
	--debug

echo "Building fleetd rpm (amd64) package..."
fleetctl package --type=rpm \
	--enable-scripts \
	--fleet-url=https://host.docker.internal:8080 \
	--enroll-secret=placeholder \
	--fleet-certificate=../osquery/fleet.crt \
	--disable-open-folder \
	--outfile=fleet-osquery_amd64.rpm \
    ${UPDATE_URL:+--update-url=$UPDATE_URL} \
    ${ORBIT_CHANNEL:+--orbit-channel=$ORBIT_CHANNEL} \
    ${DESKTOP_CHANNEL:+--desktop-channel=$DESKTOP_CHANNEL} \
    ${OSQUERYD_CHANNEL:+--osqueryd-channel=$OSQUERYD_CHANNEL} \
	--debug

echo "Building docker images..."
docker build -t fleetd-ubuntu-24.04 --platform=linux/amd64 -f ./ubuntu-24.04/Dockerfile .
docker build -t fleetd-fedora-43 --platform=linux/amd64 -f ./fedora-43/Dockerfile .
docker build -t fleetd-debian-13.4 --platform=linux/amd64 -f ./debian-13.4/Dockerfile .
