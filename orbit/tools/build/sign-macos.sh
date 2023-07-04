#!/usr/bin/env bash
set -eo pipefail

if [ -z "$CODESIGN_IDENTITY" ]
then
    echo 'Must set CODESIGN_IDENTITY in environment'
    exit 1
fi 

if [ ! -f "$1" ]
then
    echo 'First argument must be path to binary'
    exit 1
fi 

# Skip if not a macOS Mach-O executable
if ! ( file "$1" | grep Mach-O )
then
    echo 'Skip macOS signing'
    exit 0
fi

codesign -s "$CODESIGN_IDENTITY" -i com.fleetdm.orbit -f -v --timestamp --options runtime "$1"
zip -r dist/orbit-macos_darwin_all dist/orbit-macos_darwin_all
# Note that we set a 5m timeout, so this could complete without the signing being completed.
xcrun notarytool submit dist/orbit-macos_darwin_all.zip --apple-id "$AC_USERNAME" --password "$AC_PASSWORD" --team-id "$AC_TEAM_ID" --wait --timeout 5m

echo "Submitted successfully. Check notarytool submit output for status."
