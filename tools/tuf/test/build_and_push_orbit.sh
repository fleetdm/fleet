#!/bin/bash

set -x

source ./tools/tuf/test/load_orbit_version_vars.sh

GOOS=windows GOARCH=amd64 go build -o orbit.exe -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" ./orbit/cmd/orbit
./tools/tuf/test/push_target.sh windows orbit orbit.exe $ORBIT_VERSION
rm orbit.exe

GOOS=windows GOARCH=arm64 go build -o orbit-arm64.exe -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" ./orbit/cmd/orbit
./tools/tuf/test/push_target.sh windows-arm64 orbit orbit-arm64.exe $ORBIT_VERSION
rm orbit-arm64.exe

GOOS=linux GOARCH=arm64 go build -o orbit-linux-arm64 -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" ./orbit/cmd/orbit
./tools/tuf/test/push_target.sh linux-arm64 orbit orbit-linux-arm64 $ORBIT_VERSION
rm orbit-linux-arm64

GOOS=linux GOARCH=amd64 go build -o orbit-linux -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" ./orbit/cmd/orbit
./tools/tuf/test/push_target.sh linux orbit orbit-linux $ORBIT_VERSION
rm orbit-linux

CGO_ENABLED=1 ORBIT_VERSION=$ORBIT_VERSION ORBIT_COMMIT=$ORBIT_COMMIT ORBIT_BINARY_PATH=./orbit-darwin go run ./orbit/tools/build/build.go
./tools/tuf/test/push_target.sh macos orbit orbit-darwin $ORBIT_VERSION
rm orbit-darwin

echo "Done. orbit $ORBIT_VERSION built and pushed..."
