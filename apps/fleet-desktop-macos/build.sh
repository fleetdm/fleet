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

swiftc -target arm64-apple-macos13 "${SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetDesktop-arm64" "${SOURCES[@]}"
swiftc -target x86_64-apple-macos13 "${SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetDesktop-x86_64" "${SOURCES[@]}"

lipo -create \
    "$BUILD_DIR/FleetDesktop-arm64" \
    "$BUILD_DIR/FleetDesktop-x86_64" \
    -output "$MACOS_DIR/FleetDesktop"

rm "$BUILD_DIR/FleetDesktop-arm64" "$BUILD_DIR/FleetDesktop-x86_64"

# Copy Info.plist
cp "$SRC_DIR/Info.plist" "$CONTENTS_DIR/Info.plist"

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

# Keep the embedded extension's version in lockstep with the host app.
APP_SHORT_VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$CONTENTS_DIR/Info.plist")
APP_BUILD_VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleVersion" "$CONTENTS_DIR/Info.plist")
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $APP_SHORT_VERSION" "$APPEX_CONTENTS_DIR/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $APP_BUILD_VERSION" "$APPEX_CONTENTS_DIR/Info.plist"

echo "Build complete: $APP_DIR"
echo "  embedded extension: $APPEX_DIR"
echo "Run with: open \"$APP_DIR\""
