#!/usr/bin/env bash
set -eo pipefail

# This script creates a signed and notarized macOS .pkg installer for fleetctl
# It expects the signed universal binary to already exist (created by goreleaser)
# and will create the .pkg, sign it, notarize it, and upload it to the GitHub release

check_env_var() {
    if [[ -z "${!1}" ]]; then
        echo "Error: Environment variable $1 not set."
        exit 1
    fi
}

# Check required environment variables
check_env_var "APPLE_INSTALLER_CERTIFICATE"
check_env_var "APPLE_INSTALLER_CERTIFICATE_PASSWORD"
check_env_var "APPLE_USERNAME"
check_env_var "APPLE_PASSWORD"
check_env_var "APPLE_TEAM_ID"
check_env_var "KEYCHAIN_PASSWORD"
check_env_var "GITHUB_TOKEN"
check_env_var "GITHUB_REPOSITORY"
check_env_var "GITHUB_REF"

# Get version from tag (remove 'fleet-' prefix if present)
TAG_NAME="${GITHUB_REF#refs/tags/}"
if [[ "$TAG_NAME" == fleet-* ]]; then
    VERSION="${TAG_NAME#fleet-}"
else
    echo "Error: GITHUB_REF is not a tag: $GITHUB_REF"
    exit 1
fi

# Find the signed universal binary (created by goreleaser)
# Accept path as first argument, or default to dist location
if [[ -n "$1" ]]; then
    FLEETCTL_BINARY="$1"
else
    FLEETCTL_BINARY="dist/fleetctl_darwin_all/fleetctl"
fi

if [[ ! -f "$FLEETCTL_BINARY" ]]; then
    echo "Error: Signed fleetctl binary not found at $FLEETCTL_BINARY"
    echo "Available files in dist/:"
    find dist -type f 2>/dev/null || true
    exit 1
fi

echo "Found signed binary at: $FLEETCTL_BINARY"
./fleetctl --version || "$FLEETCTL_BINARY" --version

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
PACKAGE_SIGNING_IDENTITY_SHA1="D52080FD1F0941DE31346F06DA0F08AED6FACBBF"
PACKAGE_NAME="fleetctl_v${VERSION}_mac.pkg"
productsign --sign "$PACKAGE_SIGNING_IDENTITY_SHA1" \
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
    STATUS=$(echo "$NOTARIZE_OUTPUT" | grep -i "status:" | awk '{print $NF}' || echo "")
    
    echo "Notarization output:"
    echo "$NOTARIZE_OUTPUT"
    
    if [ -n "$SUBMISSION_ID" ] && [ "$STATUS" != "Accepted" ] && [ "$STATUS" != "" ]; then
        echo "⚠️  Notarization status: $STATUS"
        echo "Retrieving notarization log for details..."
        xcrun notarytool log "$SUBMISSION_ID" \
            --apple-id "$APPLE_USERNAME" \
            --password "$APPLE_PASSWORD" \
            --team-id "$APPLE_TEAM_ID" || true
        
        if [ $ATTEMPT -lt $MAX_ATTEMPTS ]; then
            echo "Retrying in 10 seconds..."
            sleep 10
            ATTEMPT=$((ATTEMPT + 1))
            continue
        else
            echo "❌ Notarization failed after $MAX_ATTEMPTS attempts. Exiting."
            exit 1
        fi
    elif [ "$STATUS" = "Accepted" ] || [ -z "$STATUS" ]; then
        # Notarization succeeded
        echo "✓ Notarization completed successfully"
        break
    fi
    
    ATTEMPT=$((ATTEMPT + 1))
done

# Staple the notarization ticket
echo "Stapling notarization ticket..."
xcrun stapler staple "$PACKAGE_NAME" || {
    echo "Warning: Stapling failed, but notarization may have succeeded"
    exit 1
}

# Clean up notarization zip
rm -f fleetctl_notarize.zip

# Move package to dist directory so it can be picked up by goreleaser or uploaded later
mkdir -p dist
mv "$PACKAGE_NAME" "dist/$PACKAGE_NAME"

echo "✓ Package created successfully: dist/$PACKAGE_NAME"

# If GITHUB_TOKEN is set and we're in a GitHub Actions environment, upload to release
if [[ -n "$GITHUB_TOKEN" ]] && [[ -n "$GITHUB_REPOSITORY" ]] && command -v gh &> /dev/null; then
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
    gh release upload "$TAG_NAME" "dist/$PACKAGE_NAME" --repo "$GITHUB_REPOSITORY" || {
        echo "Upload failed, retrying once..."
        sleep 5
        gh release upload "$TAG_NAME" "dist/$PACKAGE_NAME" --repo "$GITHUB_REPOSITORY" || {
            echo "❌ Failed to upload package after retry"
            exit 1
        }
    }
    
    echo "✓ Package uploaded successfully: dist/$PACKAGE_NAME"
fi

