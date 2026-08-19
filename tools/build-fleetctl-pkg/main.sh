#!/usr/bin/env bash
set -eo pipefail

# This script creates a signed and notarized macOS .pkg installer for fleetctl.
# It expects the signed universal binary to already exist (created by goreleaser
# or the test workflow) and will create the .pkg, sign it, notarize it, and
# optionally upload it to the GitHub release.
#
# Usage: ./main.sh <path-to-fleetctl-binary> <version>
#   version: semver like "4.72.0" or "v4.72.0" (v prefix stripped)
#
# Environment variables:
#   APPLE_INSTALLER_CERTIFICATE          - Base64-encoded installer certificate
#   APPLE_INSTALLER_CERTIFICATE_PASSWORD - Password for the installer certificate
#   APPLE_USERNAME                       - Apple ID for notarization
#   APPLE_PASSWORD                       - App-specific password for notarization
#   APPLE_TEAM_ID                        - Apple Developer Team ID
#   KEYCHAIN_PASSWORD                    - Password for the build keychain
#   SKIP_UPLOAD - set to "true" to skip GitHub release upload (default: upload)
#   GITHUB_TOKEN, GITHUB_REPOSITORY, GITHUB_REF - required if uploading to release

check_env_var() {
    if [[ -z "${!1}" ]]; then
        echo "Error: Environment variable $1 not set."
        exit 1
    fi
}

# Parse arguments
FLEETCTL_BINARY="$1"
VERSION="$2"

if [[ -z "$FLEETCTL_BINARY" ]] || [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <path-to-fleetctl-binary> <version>"
    echo "  version: semver like 4.72.0 or v4.72.0"
    exit 1
fi

if [[ ! -f "$FLEETCTL_BINARY" ]]; then
    echo "Error: Signed fleetctl binary not found at $FLEETCTL_BINARY"
    exit 1
fi

# Strip v prefix from version
VERSION="${VERSION#v}"

# Check required environment variables
check_env_var "APPLE_INSTALLER_CERTIFICATE"
check_env_var "APPLE_INSTALLER_CERTIFICATE_PASSWORD"
check_env_var "APPLE_USERNAME"
check_env_var "APPLE_PASSWORD"
check_env_var "APPLE_TEAM_ID"
check_env_var "KEYCHAIN_PASSWORD"

# Upload is enabled by default; set SKIP_UPLOAD=true to skip
if [[ "$SKIP_UPLOAD" != "true" ]]; then
    check_env_var "GITHUB_TOKEN"
    check_env_var "GITHUB_REPOSITORY"
    check_env_var "GITHUB_REF"
fi

echo "Found signed binary at: $FLEETCTL_BINARY"
"$FLEETCTL_BINARY" --version

cleanup() {
    echo "Cleaning up..."
    rm -f installer_certificate.p12
    rm -f fleetctl_unsigned.pkg
    rm -f fleetctl_notarize.zip
    rm -rf pkgroot
}

# Trap EXIT signal to call cleanup function
trap cleanup EXIT

# Import package signing keys (for .pkg)
echo "Importing package signing certificate..."
echo "$APPLE_INSTALLER_CERTIFICATE" | base64 --decode > installer_certificate.p12
security create-keychain -p "$KEYCHAIN_PASSWORD" build.keychain || true
security default-keychain -s build.keychain
security unlock-keychain -p "$KEYCHAIN_PASSWORD" build.keychain
security import installer_certificate.p12 -k build.keychain -P "$APPLE_INSTALLER_CERTIFICATE_PASSWORD" -T /usr/bin/productsign
security set-key-partition-list -S apple-tool:,apple:,productsign: -s -k "$KEYCHAIN_PASSWORD" build.keychain
security find-identity -vv
rm installer_certificate.p12

# Extract the installer signing identity from the keychain
PACKAGE_SIGNING_IDENTITY=$(security find-identity -v build.keychain | grep "Developer ID Installer" | head -1 | awk '{print $2}')
if [[ -z "$PACKAGE_SIGNING_IDENTITY" ]]; then
    echo "Error: No Developer ID Installer identity found in keychain"
    exit 1
fi
echo "Using package signing identity: $PACKAGE_SIGNING_IDENTITY"

# Create package structure
echo "Creating package structure..."
mkdir -p pkgroot/usr/local/bin
cp "$FLEETCTL_BINARY" pkgroot/usr/local/bin/fleetctl
chmod +x pkgroot/usr/local/bin/fleetctl

# Build the component package
echo "Building .pkg..."
pkgbuild \
    --root pkgroot \
    --identifier com.fleetdm.fleetctl \
    --version "$VERSION" \
    --install-location / \
    fleetctl_unsigned.pkg

# Sign the package
echo "Signing .pkg..."
PACKAGE_NAME="fleetctl_v${VERSION}_mac.pkg"
productsign --sign "$PACKAGE_SIGNING_IDENTITY" \
    fleetctl_unsigned.pkg \
    "$PACKAGE_NAME"

# Verify package signature
echo "Verifying package signature..."
pkgutil --check-signature "$PACKAGE_NAME"

# Notarize package
echo "Notarizing package..."
zip fleetctl_notarize.zip "$PACKAGE_NAME"

# Submit for notarization with retry logic
MAX_ATTEMPTS=10
ATTEMPT=1
while [ $ATTEMPT -le $MAX_ATTEMPTS ]; do
    echo "Notarization attempt $ATTEMPT of $MAX_ATTEMPTS..."
    
    NOTARIZE_OUTPUT=$(xcrun notarytool submit fleetctl_notarize.zip \
        --apple-id "$APPLE_USERNAME" \
        --password "$APPLE_PASSWORD" \
        --team-id "$APPLE_TEAM_ID" \
        --wait 2>&1) || true
    
    # Extract submission ID and status from output
    SUBMISSION_ID=$(echo "$NOTARIZE_OUTPUT" | grep -i "id:" | head -1 | awk '{print $NF}' || echo "")
    STATUS=$(echo "$NOTARIZE_OUTPUT" | grep -i "status:" | tail -1 | awk '{print $NF}' || echo "")
    
    echo "Notarization output:"
    echo "$NOTARIZE_OUTPUT"
    
    if [ "$STATUS" = "Accepted" ]; then
        echo "✓ Notarization completed successfully"
        break
    elif [ -n "$SUBMISSION_ID" ] && [ -n "$STATUS" ]; then
        echo "⚠️  Notarization status: $STATUS"
        echo "Retrieving notarization log for details..."
        xcrun notarytool log "$SUBMISSION_ID" \
            --apple-id "$APPLE_USERNAME" \
            --password "$APPLE_PASSWORD" \
            --team-id "$APPLE_TEAM_ID" || true
    else
        echo "⚠️  Could not determine notarization status (got: '$STATUS')"
    fi

    if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
        echo "Retrying in 10 seconds..."
        sleep 10
        ATTEMPT=$((ATTEMPT + 1))
        continue
    else
        echo "❌ Notarization failed after $MAX_ATTEMPTS attempts. Exiting."
        exit 1
    fi
done

# Staple the notarization ticket
echo "Stapling notarization ticket..."
xcrun stapler staple "$PACKAGE_NAME" || {
    echo "❌ Stapling failed after notarization; failing release packaging."
    exit 1
}

# Clean up notarization zip
rm -f fleetctl_notarize.zip

# Move package to dist directory so it can be picked up by goreleaser or uploaded later
mkdir -p dist
mv "$PACKAGE_NAME" "dist/$PACKAGE_NAME"

echo "✓ Package created successfully: dist/$PACKAGE_NAME"

# Upload to release unless skipped
if [[ "$SKIP_UPLOAD" == "true" ]]; then
    echo "Skipping release upload (SKIP_UPLOAD=true)"
elif [[ -n "$GITHUB_TOKEN" ]] && [[ -n "$GITHUB_REPOSITORY" ]] && command -v gh &> /dev/null; then
    if [[ "$GITHUB_REF" != refs/tags/* ]]; then
        echo "Error: GITHUB_REF is not a tag ref ($GITHUB_REF), cannot upload to release"
        exit 1
    fi
    TAG_NAME="${GITHUB_REF#refs/tags/}"
    echo "Uploading package to release $TAG_NAME..."
    
    # Wait for release to exist (goreleaser creates it as draft)
    MAX_WAIT=300  # 5 minutes max wait
    ELAPSED=0
    INTERVAL=10   # Check every 10 seconds
    
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
        echo "Attempting to upload anyway (release might be created as draft)..."
    fi
    
    echo "Uploading dist/$PACKAGE_NAME to release $TAG_NAME"
    gh release upload "$TAG_NAME" "dist/$PACKAGE_NAME" --repo "$GITHUB_REPOSITORY" --clobber || {
        echo "Upload failed, retrying once..."
        sleep 5
        gh release upload "$TAG_NAME" "dist/$PACKAGE_NAME" --repo "$GITHUB_REPOSITORY" --clobber || {
            echo "❌ Failed to upload package after retry"
            exit 1
        }
    }
    
    echo "✓ Package uploaded successfully: dist/$PACKAGE_NAME"
fi

