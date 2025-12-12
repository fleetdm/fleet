#!/bin/bash

# Constraints for orbit versioning:
# - WiX fails with "Z:\wix\main.wxs(3): error CNDL0242 : Invalid product version '10.250525.0913'. Product version
#   must have a major version less than 256, a minor version less than 256, and a build version less than 65536."
# - Cannot use "-build", WiX/Windows prefers three dots X.Y.Z.P format.
# - Must be with three parts X.Y.Z (otherwise breaks orbit semantic versioning)
ORBIT_MAJOR=$(date +"%-y") # year
ORBIT_MINOR=$(date +"%-m") # month
day=$(date +"%-d")
hour=$(date +"%-H")
minute=$(date +"%-M")
ORBIT_PATCH=$(( (day << 11) | (hour << 6) | minute )) # must fit into 16-bit number
ORBIT_VERSION="$ORBIT_MAJOR.$ORBIT_MINOR.$ORBIT_PATCH"
ORBIT_COMMIT=$(git rev-parse HEAD)

export ORBIT_MAJOR
export ORBIT_MINOR
export ORBIT_PATCH
export ORBIT_VERSION
export ORBIT_COMMIT
