#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="$SCRIPT_DIR/FleetDesktop"
BUILD_DIR="$SCRIPT_DIR/build"
APP_DIR="$BUILD_DIR/Fleet Desktop.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"

echo "Building Fleet Desktop..."

rm -rf "$BUILD_DIR"
mkdir -p "$MACOS_DIR"

SOURCES=(
    "$SRC_DIR/FleetService.swift"
    "$SRC_DIR/BrowserWindow.swift"
    "$SRC_DIR/FleetDesktopApp.swift"
)
SDK="$(xcrun --show-sdk-path)"
SWIFT_FLAGS=(-sdk "$SDK" -parse-as-library -O)

# Build for arm64
swiftc -target arm64-apple-macos13 "${SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetDesktop-arm64" "${SOURCES[@]}"

# Build for x86_64
swiftc -target x86_64-apple-macos13 "${SWIFT_FLAGS[@]}" \
    -o "$BUILD_DIR/FleetDesktop-x86_64" "${SOURCES[@]}"

# Create universal binary
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

echo "Build complete: $APP_DIR"
echo "Run with: open \"$APP_DIR\""
