#!/bin/bash

# Orbit (test) version number are currently constrained by this error from WiX:
# "Z:\wix\main.wxs(3): error CNDL0242 : Invalid product version '10.250525.0913'. Product version
# must have a major version less than 256, a minor version less than 256, and a build version less than 65536."
ORBIT_MAJOR=$(date +"%y")
ORBIT_MINOR=$(date +"%m")
ORBIT_PATCH=$(date +"%d")
ORBIT_BUILD=$(date +"%H%M")
ORBIT_VERSION="$ORBIT_MAJOR.$ORBIT_MINOR.$ORBIT_PATCH.$ORBIT_BUILD"
ORBIT_COMMIT=$(git rev-parse HEAD)

export ORBIT_MAJOR
export ORBIT_MINOR
export ORBIT_PATCH
export ORBIT_BUILD
export ORBIT_VERSION
export ORBIT_COMMIT