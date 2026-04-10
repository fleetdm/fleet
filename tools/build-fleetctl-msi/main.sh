#!/usr/bin/env bash
set -eo pipefail

# This script builds a signed Windows .msi installer for fleetctl.
# It expects to run on a Windows GitHub Actions runner with Git Bash,
# with DigiCert KeyLocker already configured (env vars set, KSP installed,
# certs synced) and WiX 3.14 tools on PATH.
#
# Usage: ./main.sh <path-to-fleetctl.exe> <version> <arch>
#   version: semver like "4.72.0" or "v4.72.0" (v prefix stripped)
#   arch:    "amd64" or "arm64"
#
# Environment variables:
#   DIGICERT_KEYLOCKER_CERTIFICATE_FINGERPRINT - SHA1 fingerprint for signtool
#   SKIP_UPLOAD - set to "true" to skip GitHub release upload (default: skip)
#   GITHUB_TOKEN, GITHUB_REPOSITORY, GITHUB_REF - required if uploading to release

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

check_env_var() {
    if [[ -z "${!1}" ]]; then
        echo "Error: Environment variable $1 not set."
        exit 1
    fi
}

# Parse arguments
FLEETCTL_EXE="$1"
VERSION="$2"
ARCH="${3:-amd64}"

if [[ -z "$FLEETCTL_EXE" ]] || [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <path-to-fleetctl.exe> <version> [arch]"
    echo "  version: semver like 4.72.0 or v4.72.0"
    echo "  arch:    amd64 (default) or arm64"
    exit 1
fi

if [[ ! -f "$FLEETCTL_EXE" ]]; then
    echo "Error: fleetctl.exe not found at $FLEETCTL_EXE"
    exit 1
fi

# Check required env var for signing
check_env_var "DIGICERT_KEYLOCKER_CERTIFICATE_FINGERPRINT"

# Strip v prefix from version
VERSION="${VERSION#v}"

# Validate version format (WiX requires X.Y.Z or X.Y.Z.W, each component 0-65535)
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?$ ]]; then
    echo "Error: Version '$VERSION' is not a valid MSI version (expected X.Y.Z or X.Y.Z.W)"
    exit 1
fi

# Map Go arch to WiX candle arch
case "$ARCH" in
    amd64) WIX_ARCH="x64" ;;
    arm64) WIX_ARCH="arm64" ;;
    *)
        echo "Error: Unsupported architecture '$ARCH' (expected amd64 or arm64)"
        exit 1
        ;;
esac

MSI_NAME="fleetctl_v${VERSION}_windows_${ARCH}.msi"
echo "Building $MSI_NAME..."

# Create working directory
WORKDIR="$(mktemp -d)"
# Convert to Windows path for native Windows tools (signtool, candle, light)
WORKDIR_WIN="$(cygpath -w "$WORKDIR")"
cleanup() {
    echo "Cleaning up..."
    rm -rf "$WORKDIR"
}
trap cleanup EXIT

# Set up working directory structure
mkdir -p "$WORKDIR/root"
cp "$FLEETCTL_EXE" "$WORKDIR/root/fleetctl.exe"

# Substitute version placeholder in WiX template
sed "s/__VERSION__/$VERSION/g" "$SCRIPT_DIR/fleetctl.wxs" > "$WORKDIR/main.wxs"

# Sign fleetctl.exe before packaging into MSI
# MSYS_NO_PATHCONV=1 prevents Git Bash from converting /flags to Windows paths
# Windows-native tools need Windows-style paths (WORKDIR_WIN)
echo "Signing fleetctl.exe..."
MSYS_NO_PATHCONV=1 signtool.exe sign /v /sha1 "$DIGICERT_KEYLOCKER_CERTIFICATE_FINGERPRINT" \
    /tr http://timestamp.digicert.com /td SHA256 /fd SHA256 \
    /d "fleetctl by Fleet Device Management" /du "https://fleetdm.com" \
    "$WORKDIR_WIN\\root\\fleetctl.exe"
MSYS_NO_PATHCONV=1 signtool.exe verify /v /pa "$WORKDIR_WIN\\root\\fleetctl.exe"
echo "fleetctl.exe signed successfully"

# Compile WiX source
echo "Running candle (WiX compiler)..."
candle.exe "$WORKDIR_WIN\\main.wxs" -arch "$WIX_ARCH" -out "$WORKDIR_WIN\\main.wixobj"

# Link to create MSI
echo "Running light (WiX linker)..."
light.exe "$WORKDIR_WIN\\main.wixobj" -b "$WORKDIR_WIN" -out "$WORKDIR_WIN\\$MSI_NAME" -sval

# Sign the MSI
echo "Signing MSI..."
MSYS_NO_PATHCONV=1 signtool.exe sign /v /sha1 "$DIGICERT_KEYLOCKER_CERTIFICATE_FINGERPRINT" \
    /tr http://timestamp.digicert.com /td SHA256 /fd SHA256 \
    /d "fleetctl by Fleet Device Management" /du "https://fleetdm.com" \
    "$WORKDIR_WIN\\$MSI_NAME"
MSYS_NO_PATHCONV=1 signtool.exe verify /v /pa "$WORKDIR_WIN\\$MSI_NAME"
echo "MSI signed successfully"

# Copy to output location
mkdir -p dist
cp "$WORKDIR/$MSI_NAME" "dist/$MSI_NAME"
echo "Package created: dist/$MSI_NAME"

# Upload to release unless skipped
if [[ "$SKIP_UPLOAD" == "true" ]]; then
    echo "Skipping release upload (SKIP_UPLOAD=true)"
elif [[ -n "$GITHUB_TOKEN" ]] && [[ -n "$GITHUB_REPOSITORY" ]] && [[ -n "$GITHUB_REF" ]] && command -v gh &> /dev/null; then
    if [[ "$GITHUB_REF" != refs/tags/* ]]; then
        echo "Error: GITHUB_REF is not a tag ref ($GITHUB_REF), cannot upload to release"
        exit 1
    fi
    TAG_NAME="${GITHUB_REF#refs/tags/}"
    echo "Uploading package to release $TAG_NAME..."

    # Wait for release to exist (goreleaser creates it as draft)
    MAX_WAIT=300
    ELAPSED=0
    INTERVAL=10

    while [ $ELAPSED -lt $MAX_WAIT ]; do
        if gh release view "$TAG_NAME" --repo "$GITHUB_REPOSITORY" &>/dev/null; then
            echo "Release found, uploading package..."
            break
        fi
        echo "Waiting for release to be created... (${ELAPSED}s/${MAX_WAIT}s)"
        sleep $INTERVAL
        ELAPSED=$((ELAPSED + INTERVAL))
    done

    if [ $ELAPSED -ge $MAX_WAIT ]; then
        echo "Warning: Release not found after waiting ${MAX_WAIT}s"
        echo "Attempting upload anyway..."
    fi

    echo "Uploading dist/$MSI_NAME to release $TAG_NAME"
    gh release upload "$TAG_NAME" "dist/$MSI_NAME" --repo "$GITHUB_REPOSITORY" --clobber || {
        echo "Upload failed, retrying once..."
        sleep 5
        gh release upload "$TAG_NAME" "dist/$MSI_NAME" --repo "$GITHUB_REPOSITORY" --clobber || {
            echo "Failed to upload package after retry"
            exit 1
        }
    }

    echo "Package uploaded successfully: dist/$MSI_NAME"
else
    echo "Skipping release upload (GITHUB_TOKEN or gh not available)"
fi
