#!/bin/bash

set -ex

# This script generates fleet-osquery packages for all supported platforms
# using the specified TUF server.

# Input:
# Values for generating a package for a macOS host:
# PKG_FLEET_URL: Fleet server URL.
# PKG_TUF_URL: URL of the TUF server.
#
# Values for generating a package for an Ubuntu host:
# DEB_FLEET_URL: Fleet server URL.
# DEB_TUF_URL: URL of the TUF server.
#
# Values for generating a package for a CentOS host:
# RPM_FLEET_URL: Fleet server URL.
# RPM_TUF_URL: URL of the TUF server.
#
# Values for generating a package for a Windows host:
# MSI_FLEET_URL: Fleet server URL.
# MSI_TUF_URL: URL of the TUF server.
#
# ENROLL_SECRET: Fleet server enroll secret.
# ROOT_KEYS: TUF repository root keys.
# FLEET_DESKTOP: Whether to build with Fleet Desktop support. 
# FLEET_CERTIFICATE: Whether to use a custom certificate bundle. If not set, then --insecure mode is used.

TLS_FLAG="--insecure"
if [ -n "$FLEET_CERTIFICATE" ]; then
    TLS_FLAG="--fleet-certificate=./tools/osquery/fleet.crt"
fi

if [ -n "$GENERATE_PKG" ]; then
    echo "Generating pkg..."
    ./build/fleetctl package \
        --type=pkg \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$PKG_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${TLS_FLAG} \
        --debug \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        --update-url=$PKG_TUF_URL
fi

if [ -n "$GENERATE_DEB" ]; then
    echo "Generating deb..."
    ./build/fleetctl package \
        --type=deb \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$DEB_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${TLS_FLAG} \
        --debug \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        --update-url=$DEB_TUF_URL
fi

if [ -n "$GENERATE_RPM" ]; then
    echo "Generating rpm..."
    ./build/fleetctl package \
        --type=rpm \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$RPM_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${TLS_FLAG} \
        --debug \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        --update-url=$RPM_TUF_URL
fi

if [ -n "$GENERATE_MSI" ]; then
    echo "Generating msi..."
    ./build/fleetctl package \
        --type=msi \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$MSI_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${TLS_FLAG} \
        --debug \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        --update-url=$MSI_TUF_URL
fi

echo "Packages generated."

if [[ $OSTYPE == 'darwin'* && -n "$INSTALL_PKG" ]]; then
    sudo installer -pkg fleet-osquery.pkg -target /
fi