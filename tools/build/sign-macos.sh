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

echo "Signed successfully"
