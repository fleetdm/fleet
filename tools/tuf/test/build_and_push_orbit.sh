#!/bin/bash

set -x

source ./tools/tuf/test/load_orbit_version_vars.sh

# Mirrors orbit/goreleaser-windows*.yml: generate the versioninfo resource, then build with the release flags.
build_windows_orbit() {
    local arch=$1 output=$2
    go run ./orbit/tools/build/build-windows.go -version "$ORBIT_VERSION" -input ./orbit/cmd/orbit -resource=true -arch "$arch"
    CGO_ENABLED=0 GOOS=windows GOARCH=$arch go build -trimpath -o "$output" -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" ./orbit/cmd/orbit
    rm -f ./orbit/cmd/orbit/resource_windows.syso ./orbit/cmd/orbit/manifest.xml
}

build_windows_orbit amd64 orbit.exe
./tools/tuf/test/push_target.sh windows orbit orbit.exe $ORBIT_VERSION
rm orbit.exe

build_windows_orbit arm64 orbit-arm64.exe
./tools/tuf/test/push_target.sh windows-arm64 orbit orbit-arm64.exe $ORBIT_VERSION
rm orbit-arm64.exe

GOOS=linux GOARCH=arm64 go build -o orbit-linux-arm64 -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" ./orbit/cmd/orbit
./tools/tuf/test/push_target.sh linux-arm64 orbit orbit-linux-arm64 $ORBIT_VERSION
rm orbit-linux-arm64

GOOS=linux GOARCH=amd64 go build -o orbit-linux -ldflags="-s -w -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Version=$ORBIT_VERSION -X github.com/fleetdm/fleet/v4/orbit/pkg/build.Commit=$ORBIT_COMMIT" ./orbit/cmd/orbit
./tools/tuf/test/push_target.sh linux orbit orbit-linux $ORBIT_VERSION
rm orbit-linux

CGO_ENABLED=1 ORBIT_VERSION=$ORBIT_VERSION ORBIT_COMMIT=$ORBIT_COMMIT ORBIT_BINARY_PATH=./orbit-darwin go run ./orbit/tools/build/build.go
# Without a codesign identity, ad-hoc sign orbit with its release identifier, which the Microsoft
# SSO extension's AppAllowList matches for the Entra device token.
if [ -z "$CODESIGN_IDENTITY" ]; then
    codesign -s - -i com.fleetdm.orbit -f orbit-darwin
fi
./tools/tuf/test/push_target.sh macos orbit orbit-darwin $ORBIT_VERSION
rm orbit-darwin

echo "Done. orbit $ORBIT_VERSION built and pushed..."
