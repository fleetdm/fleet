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
# INSECURE: Whether to use the --insecure flag.
# USE_FLEET_SERVER_CERTIFICATE: Whether to use a custom certificate bundle.
# USE_UPDATE_SERVER_CERTIFICATE: Whether to use a custom certificate bundle.
# FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST: Alternative host:port to use for the Fleet Desktop browser URLs.
# DEBUG: Whether or not to build the package with --debug.

if [ -n "$GENERATE_PKG" ]; then
    echo "Generating pkg..."
    ./build/fleetctl package \
        --type=pkg \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$PKG_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${USE_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=./tools/osquery/fleet.crt} \
        ${USE_UPDATE_SERVER_CERTIFICATE:+--update-tls-certificate=./tools/osquery/fleet.crt} \
        ${INSECURE:+--insecure} \
        ${DEBUG:+--debug} \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST:+--fleet-desktop-alternative-browser-host=$FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST} \
        --update-url=$PKG_TUF_URL \
        --disable-keystore \
        --enable-scripts
fi

if [ -n "$GENERATE_DEB" ]; then
    echo "Generating deb..."
    ./build/fleetctl package \
        --type=deb \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$DEB_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${USE_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=./tools/osquery/fleet.crt} \
        ${USE_UPDATE_SERVER_CERTIFICATE:+--update-tls-certificate=./tools/osquery/fleet.crt} \
        ${INSECURE:+--insecure} \
        ${DEBUG:+--debug} \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST:+--fleet-desktop-alternative-browser-host=$FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST} \
        --update-url=$DEB_TUF_URL \
        --enable-scripts
fi

if [ -n "$GENERATE_RPM" ]; then
    echo "Generating rpm..."
    ./build/fleetctl package \
        --type=rpm \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$RPM_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${USE_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=./tools/osquery/fleet.crt} \
        ${USE_UPDATE_SERVER_CERTIFICATE:+--update-tls-certificate=./tools/osquery/fleet.crt} \
        ${INSECURE:+--insecure} \
        ${DEBUG:+--debug} \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST:+--fleet-desktop-alternative-browser-host=$FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST} \
        --update-url=$RPM_TUF_URL \
        --enable-scripts
fi

if [ -n "$GENERATE_MSI" ]; then
    echo "Generating msi..."
    ./build/fleetctl package \
        --type=msi \
        ${FLEET_DESKTOP:+--fleet-desktop} \
        --fleet-url=$MSI_FLEET_URL \
        --enroll-secret=$ENROLL_SECRET \
        ${USE_FLEET_SERVER_CERTIFICATE:+--fleet-certificate=./tools/osquery/fleet.crt} \
        ${USE_UPDATE_SERVER_CERTIFICATE:+--update-tls-certificate=./tools/osquery/fleet.crt} \
        ${INSECURE:+--insecure} \
        ${DEBUG:+--debug} \
        --update-roots="$ROOT_KEYS" \
        --update-interval=10s \
        --disable-open-folder \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_FLEET_CLIENT_CERTIFICATE:+--fleet-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-certificate=./tools/test-orbit-mtls/client.crt} \
        ${USE_UPDATE_CLIENT_CERTIFICATE:+--update-tls-client-key=./tools/test-orbit-mtls/client.key} \
        ${FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST:+--fleet-desktop-alternative-browser-host=$FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST} \
        --update-url=$MSI_TUF_URL \
        --enable-scripts
fi

echo "Packages generated."

if [[ $OSTYPE == 'darwin'* && -n "$INSTALL_PKG" ]]; then
    sudo installer -pkg fleet-osquery.pkg -target /
fi