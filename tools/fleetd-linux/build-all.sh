#!/bin/bash

set -euo pipefail

script_dir=$(dirname -- "$(readlink -f -- "$BASH_SOURCE")")
cd "$script_dir"

FLEET_URL=${FLEET_URL:-https://host.docker.internal:8080}

using_localhost() {
    local url=$1
    if [[ "$url" = "https://localhost:"* || "$url" = "https://127.0.0.1:"* || "$url" = "https://host.docker.internal:"* ]]; then
        return 0 # true (exit code 0)
    else
        return 1 # false (exit code 1)
    fi
}

USE_LOCALHOST_FLEET_SERVER_CERTIFICATE=
if using_localhost "$FLEET_URL"; then
    USE_LOCALHOST_FLEET_SERVER_CERTIFICATE=1
fi

echo "Building fleetd deb (amd64) package..."
fleetctl package --type=deb \
	--enable-scripts \
	--fleet-url="$FLEET_URL" \
	--enroll-secret=placeholder \
	${USE_LOCALHOST_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=../osquery/fleet.crt} \
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
	--fleet-url="$FLEET_URL" \
	--enroll-secret=placeholder \
	${USE_LOCALHOST_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=../osquery/fleet.crt} \
	--disable-open-folder \
	--outfile=fleet-osquery_amd64.rpm \
    ${UPDATE_URL:+--update-url=$UPDATE_URL} \
    ${ORBIT_CHANNEL:+--orbit-channel=$ORBIT_CHANNEL} \
    ${DESKTOP_CHANNEL:+--desktop-channel=$DESKTOP_CHANNEL} \
    ${OSQUERYD_CHANNEL:+--osqueryd-channel=$OSQUERYD_CHANNEL} \
	--debug

echo "Building fleetd pkg.tar.zst (amd64) package..."
fleetctl package --type=pkg.tar.zst \
	--enable-scripts \
	--fleet-url="$FLEET_URL" \
	--enroll-secret=placeholder \
	${USE_LOCALHOST_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=../osquery/fleet.crt} \
	--disable-open-folder \
	--outfile=fleet-osquery_amd64.pkg.tar.zst \
    ${UPDATE_URL:+--update-url=$UPDATE_URL} \
    ${ORBIT_CHANNEL:+--orbit-channel=$ORBIT_CHANNEL} \
    ${DESKTOP_CHANNEL:+--desktop-channel=$DESKTOP_CHANNEL} \
    ${OSQUERYD_CHANNEL:+--osqueryd-channel=$OSQUERYD_CHANNEL} \
	--debug

echo "Building docker images..."
docker build -t fleetd-ubuntu-24.04 --platform=linux/amd64 -f ./ubuntu-24.04/Dockerfile .
docker build -t fleetd-fedora-43 --platform=linux/amd64 -f ./fedora-43/Dockerfile .
docker build -t fleetd-debian-13.4 --platform=linux/amd64 -f ./debian-13.4/Dockerfile .
docker build -t fleetd-cachyos --platform=linux/amd64 -f ./cachyos/Dockerfile .
