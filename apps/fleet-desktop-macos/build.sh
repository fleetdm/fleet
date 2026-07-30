#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="$SCRIPT_DIR/FleetDesktop"
EXT_SRC_DIR="$SCRIPT_DIR/FleetPSSOExtension"
BUILD_DIR="$SCRIPT_DIR/build"
APP_DIR="$BUILD_DIR/Fleet Desktop.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
APPEX_DIR="$CONTENTS_DIR/PlugIns/FleetPSSOExtension.appex"
APPEX_CONTENTS_DIR="$APPEX_DIR/Contents"
APPEX_MACOS_DIR="$APPEX_CONTENTS_DIR/MacOS"

# --- Identity & signing (all optional; production defaults) -----------------
# For local development a contributor signs under their own (non-production)
# Apple Developer team, which owns different bundle IDs than Fleet's. Override
# these to produce a dev-signed bundle. Left unset, the script behaves exactly
# as before: a compile-only bundle that CI signs separately with Fleet's certs.
# See the "Local PSSO development" contributor guide.
APP_BUNDLE_ID="${APP_BUNDLE_ID:-com.fleetdm.fleet-desktop}"
EXT_BUNDLE_ID="${EXT_BUNDLE_ID:-com.fleetdm.fleet-desktop.pssoextension}"
TEAM_ID="${TEAM_ID:-8VBZ3948LU}"
SIGNING_IDENTITY="${SIGNING_IDENTITY:-}" # e.g. "Apple Development: you@example.com (XXXXXXXXXX)"; empty => no signing
APP_PROFILE="${APP_PROFILE:-}"           # path to the host-app .provisionprofile (required to sign)
EXT_PROFILE="${EXT_PROFILE:-}"           # path to the extension .provisionprofile (required to sign)

echo "Building Fleet Desktop..."

rm -rf "$BUILD_DIR"
mkdir -p "$MACOS_DIR"

SDK="$(xcrun --show-sdk-path)"

# --- Host app -------------------------------------------------------------
SOURCES=(
    "$SRC_DIR/FleetService.swift"
    "$SRC_DIR/BrowserWindow.swift"
    "$SRC_DIR/FleetDesktopApp.swift"
)
SWIFT_FLAGS=(-sdk "$SDK" -parse-as-library -O)

# Build both architectures in parallel. Collect both exit statuses before
# failing — a bare `wait PID` under set -e exits on the first failure and
# leaves the other swiftc running in the background.
swiftc -target arm64-apple-macos13 "${SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetDesktop-arm64" "${SOURCES[@]}" &
ARM64_PID=$!

swiftc -target x86_64-apple-macos13 "${SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetDesktop-x86_64" "${SOURCES[@]}" &
X86_64_PID=$!

ARM64_STATUS=0
X86_64_STATUS=0
wait "$ARM64_PID" || ARM64_STATUS=$?
wait "$X86_64_PID" || X86_64_STATUS=$?
if [ "$ARM64_STATUS" -ne 0 ] || [ "$X86_64_STATUS" -ne 0 ]; then
    echo "swiftc failed (arm64 exit $ARM64_STATUS, x86_64 exit $X86_64_STATUS)" >&2
    exit 1
fi

lipo -create \
    "$BUILD_DIR/FleetDesktop-arm64" \
    "$BUILD_DIR/FleetDesktop-x86_64" \
    -output "$MACOS_DIR/FleetDesktop"

rm "$BUILD_DIR/FleetDesktop-arm64" "$BUILD_DIR/FleetDesktop-x86_64"

# Copy Info.plist
cp "$SRC_DIR/Info.plist" "$CONTENTS_DIR/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleIdentifier $APP_BUNDLE_ID" "$CONTENTS_DIR/Info.plist"

# Copy app icon and Fleet logo into Resources
mkdir -p "$CONTENTS_DIR/Resources"
cp "$SRC_DIR/AppIcon.icns" "$CONTENTS_DIR/Resources/AppIcon.icns"
if [ -f "$SRC_DIR/fleet-logo.png" ]; then
    cp "$SRC_DIR/fleet-logo.png" "$CONTENTS_DIR/Resources/fleet-logo.png"
fi

# --- Platform SSO extension (.appex) --------------------------------------
# Built as a Foundation app extension: no main(), entry point is
# NSExtensionMain (the principal class comes from the appex Info.plist).
# -module-name must match the NSExtensionPrincipalClass module prefix.
echo "Building Fleet PSSO extension..."
mkdir -p "$APPEX_MACOS_DIR"

EXT_SOURCES=(
    "$EXT_SRC_DIR/AuthenticationViewController.swift"
    "$EXT_SRC_DIR/AuthenticationViewController+PSSO.swift"
    "$EXT_SRC_DIR/AuthenticationViewController+Shared.swift"
    "$EXT_SRC_DIR/AuthenticationViewController+Networking.swift"
)
EXT_SWIFT_FLAGS=(
    -sdk "$SDK" -parse-as-library -O
    -module-name FleetPSSOExtension
    -framework AuthenticationServices -framework IOKit
    -Xlinker -e -Xlinker _NSExtensionMain
)

swiftc -target arm64-apple-macos14 "${EXT_SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetPSSOExtension-arm64" "${EXT_SOURCES[@]}"
swiftc -target x86_64-apple-macos14 "${EXT_SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetPSSOExtension-x86_64" "${EXT_SOURCES[@]}"

lipo -create \
    "$BUILD_DIR/FleetPSSOExtension-arm64" \
    "$BUILD_DIR/FleetPSSOExtension-x86_64" \
    -output "$APPEX_MACOS_DIR/FleetPSSOExtension"

rm "$BUILD_DIR/FleetPSSOExtension-arm64" "$BUILD_DIR/FleetPSSOExtension-x86_64"

cp "$EXT_SRC_DIR/Info.plist" "$APPEX_CONTENTS_DIR/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleIdentifier $EXT_BUNDLE_ID" "$APPEX_CONTENTS_DIR/Info.plist"

# Keep the embedded extension's version in lockstep with the host app.
APP_SHORT_VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$CONTENTS_DIR/Info.plist")
APP_BUILD_VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleVersion" "$CONTENTS_DIR/Info.plist")
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $APP_SHORT_VERSION" "$APPEX_CONTENTS_DIR/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $APP_BUILD_VERSION" "$APPEX_CONTENTS_DIR/Info.plist"

# --- Optional code signing (local development) ------------------------------
# The associated-domains entitlements are Apple-managed; codesign only honors
# them with a matching provisioning profile embedded in the bundle. This mirrors
# CI's inside-out signing (extension first, then host app), but with the dev
# team's identity, bundle IDs, and profiles.
if [ -n "$SIGNING_IDENTITY" ]; then
    echo "Signing under team $TEAM_ID with identity: $SIGNING_IDENTITY"
    if [ -z "$APP_PROFILE" ] || [ -z "$EXT_PROFILE" ]; then
        echo "ERROR: SIGNING_IDENTITY is set but APP_PROFILE / EXT_PROFILE are not." >&2
        echo "Signing without an embedded profile leaves the restricted associated-domains" >&2
        echo "entitlements unauthorized, so Platform SSO will not engage." >&2
        exit 1
    fi

    # Substitute the dev team + bundle IDs into throwaway copies of the committed
    # entitlements so the sealed application-identifier matches the signed binary.
    APP_ENT="$BUILD_DIR/FleetDesktop.dev.entitlements"
    EXT_ENT="$BUILD_DIR/FleetPSSOExtension.dev.entitlements"
    cp "$SRC_DIR/FleetDesktop.entitlements" "$APP_ENT"
    cp "$EXT_SRC_DIR/FleetPSSOExtension.entitlements" "$EXT_ENT"
    /usr/libexec/PlistBuddy -c "Set :com.apple.application-identifier $TEAM_ID.$APP_BUNDLE_ID" "$APP_ENT"
    /usr/libexec/PlistBuddy -c "Set :com.apple.developer.team-identifier $TEAM_ID" "$APP_ENT"
    /usr/libexec/PlistBuddy -c "Set :com.apple.application-identifier $TEAM_ID.$EXT_BUNDLE_ID" "$EXT_ENT"
    /usr/libexec/PlistBuddy -c "Set :com.apple.developer.team-identifier $TEAM_ID" "$EXT_ENT"

    cp "$EXT_PROFILE" "$APPEX_CONTENTS_DIR/embedded.provisionprofile"
    cp "$APP_PROFILE" "$CONTENTS_DIR/embedded.provisionprofile"

    codesign --force --options runtime --sign "$SIGNING_IDENTITY" \
        --entitlements "$EXT_ENT" "$APPEX_DIR"
    codesign --force --options runtime --sign "$SIGNING_IDENTITY" \
        --entitlements "$APP_ENT" "$APP_DIR"
    codesign --verify --deep --strict --verbose=2 "$APP_DIR"
fi

echo "Build complete: $APP_DIR"
echo "  embedded extension: $APPEX_DIR"
echo "Run with: open \"$APP_DIR\""
