#!/bin/bash

# This script is to build the tuf repo and package it and the file-server into a docker container to deploy.

if [ $# -lt 2 ]; then
  echo 1>&2 "$0: not enough arguments"
  exit 2
elif [ $# -gt 2 ]; then
  echo 1>&2 "$0: too many arguments"
  exit 2
fi

BASE_TUF_URL=$1
BASE_FLEET_URL=$2

rm -rf test_tuf desktop.tar.gz

SYSTEMS="macos windows linux linux-arm64" \
PKG_FLEET_URL=$BASE_FLEET_URL \
PKG_TUF_URL=$BASE_TUF_URL \
DEB_FLEET_URL=$BASE_FLEET_URL \
DEB_TUF_URL=$BASE_TUF_URL \
RPM_FLEET_URL=$BASE_FLEET_URL \
RPM_TUF_URL=$BASE_TUF_URL \
MSI_FLEET_URL=$BASE_FLEET_URL \
MSI_TUF_URL=$BASE_TUF_URL \
GENERATE_PKG=1 \
GENERATE_DEB=1 \
GENERATE_RPM=1 \
GENERATE_MSI=1 \
ENROLL_SECRET=6/EzU/+jPkxfTamWnRv1+IJsO4T9Etju \
FLEET_DESKTOP=1 \
USE_FLEET_SERVER_CERTIFICATE=1 \
SKIP_SERVER=1 \
./tools/tuf/test/main.sh

rm -f file-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o file-server ./tools/file-server
chmod +x ./file-server

TAG=testing
docker build -t $TAG -f tools/tuf/test/Dockerfile .
