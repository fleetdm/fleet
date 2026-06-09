#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$SCRIPT_DIR/build"
APP_DIR="$BUILD_DIR/Fleet Desktop.app"
PKG_DIR="$BUILD_DIR/pkg"
DIST_DIR="$BUILD_DIR/dist"

# Only build if app doesn't exist or if FORCE_REBUILD is set
if [ ! -d "$APP_DIR" ] || [ "${FORCE_REBUILD:-}" = "1" ]; then
  echo "Building Fleet Desktop app..."
  bash "$SCRIPT_DIR/build.sh"
else
  echo "Using existing app at $APP_DIR (skip rebuild)"
  # Verify the app is signed if it exists
  if codesign --verify "$APP_DIR" &>/dev/null; then
    echo "App is already signed, using as-is"
  else
    echo "Warning: App exists but is not signed"
  fi
fi

echo "Preparing package structure..."
rm -rf "$PKG_DIR" "$DIST_DIR"
mkdir -p "$PKG_DIR/Applications"
# Use ditto to preserve extended attributes and signatures
ditto "$APP_DIR" "$PKG_DIR/Applications/Fleet Desktop.app"

# Create preinstall script to check MDM and quit the app if running
cat > "$PKG_DIR/preinstall" << 'PREINSTALL_EOF'
#!/bin/bash
# Preinstall script: verify MDM enrollment, gracefully quit Fleet Desktop
# if it is running, and track its state so postinstall can relaunch it.

MDM_PLIST="/Library/Managed Preferences/com.fleetdm.fleetd.config.plist"
if [ ! -f "$MDM_PLIST" ]; then
    echo "ERROR: Fleet Desktop requires an MDM-enabled Mac." >&2
    echo "The managed preferences file was not found at: $MDM_PLIST" >&2
    echo "Please enroll this device via MDM before installing Fleet Desktop." >&2
    exit 1
fi

BUNDLE_ID="com.fleetdm.fleet-desktop"
RUNNING_FLAG="/tmp/.fleet_desktop_was_running"

# Clean up any stale flag from a previous install
rm -f "$RUNNING_FLAG"

# Check if a GUI user is logged in (osascript won't work otherwise)
console_user=$(stat -f "%Su" /dev/console 2>/dev/null || echo "root")
if [[ "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    # No GUI session — nothing to quit or relaunch
    exit 0
fi

# Check if the app is running
if osascript -e "application id \"$BUNDLE_ID\" is running" 2>/dev/null | grep -qi "true"; then
    # Mark that it was running so postinstall can relaunch
    touch "$RUNNING_FLAG"

    # Attempt graceful quit
    osascript -e "tell application id \"$BUNDLE_ID\" to quit" 2>/dev/null || true

    # Wait up to 10 seconds for the process to exit
    for i in $(seq 1 10); do
        if ! pgrep -f "$BUNDLE_ID" >/dev/null 2>&1 && ! pgrep -x "FleetDesktop" >/dev/null 2>&1; then
            break
        fi
        sleep 1
    done

    # Force kill if still running
    if pgrep -x "FleetDesktop" >/dev/null 2>&1; then
        pkill -x "FleetDesktop" 2>/dev/null || true
        sleep 1
    fi
fi

exit 0
PREINSTALL_EOF

chmod +x "$PKG_DIR/preinstall"

# Create postinstall script to set ownership/permissions and relaunch if needed
cat > "$PKG_DIR/postinstall" << 'POSTINSTALL_EOF'
#!/bin/bash
# Postinstall script: set ownership/permissions and relaunch if the app was running

APP_PATH="/Applications/Fleet Desktop.app"
BUNDLE_ID="com.fleetdm.fleet-desktop"
RUNNING_FLAG="/tmp/.fleet_desktop_was_running"

# Set ownership to root:admin
chown -R root:admin "$APP_PATH"

# Set permissions to 755
chmod -R 755 "$APP_PATH"

# Ensure the executable has proper permissions
chmod +x "$APP_PATH/Contents/MacOS/FleetDesktop"

# Relaunch the app if it was running before the install
if [ -f "$RUNNING_FLAG" ]; then
    rm -f "$RUNNING_FLAG"

    # Check if a GUI user is logged in
    console_user=$(stat -f "%Su" /dev/console 2>/dev/null || echo "root")
    if [[ "$console_user" != "root" && "$console_user" != "loginwindow" ]]; then
        # Open the app as the console user (not as root)
        sudo -u "$console_user" open "$APP_PATH" 2>/dev/null || true
    fi
fi

exit 0
POSTINSTALL_EOF

chmod +x "$PKG_DIR/postinstall"

# Extract version from Info.plist
VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" "$APP_DIR/Contents/Info.plist")
PKG_NAME="fleet_desktop-v${VERSION}.pkg"

echo "Building component package..."
mkdir -p "$DIST_DIR"
COMPONENT_PKG="$BUILD_DIR/fleet-desktop-component.pkg"
pkgbuild \
    --root "$PKG_DIR/Applications" \
    --scripts "$PKG_DIR" \
    --identifier com.fleetdm.fleet-desktop \
    --version "${VERSION}" \
    --install-location /Applications \
    "$COMPONENT_PKG"

# Create distribution XML for custom installer title
DIST_XML="$BUILD_DIR/distribution.xml"
cat > "$DIST_XML" << DIST_EOF
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="2">
    <title>Fleet Desktop v${VERSION}</title>
    <options customize="never" require-scripts="false" hostArchitectures="x86_64,arm64"/>
    <installation-check script="mdm_check()"/>
    <script>
function mdm_check() {
    if (system.files.fileExistsAtPath('/Library/Managed Preferences/com.fleetdm.fleetd.config.plist')) {
        return true;
    }
    my.result.title = 'Installation Failed';
    my.result.message = 'Fleet Desktop requires an MDM-enabled Mac. Please enroll this device via MDM before installing Fleet Desktop.';
    my.result.type = 'Fatal';
    return false;
}
    </script>
    <choices-outline>
        <line choice="default"/>
    </choices-outline>
    <choice id="default" title="Fleet Desktop">
        <pkg-ref id="com.fleetdm.fleet-desktop"/>
    </choice>
    <pkg-ref id="com.fleetdm.fleet-desktop" version="${VERSION}" onConclusion="none">fleet-desktop-component.pkg</pkg-ref>
</installer-gui-script>
DIST_EOF

echo "Building product package with custom installer title..."
productbuild \
    --distribution "$DIST_XML" \
    --package-path "$BUILD_DIR" \
    "$DIST_DIR/$PKG_NAME"

# Clean up component package
rm -f "$COMPONENT_PKG"

echo "Package created: $DIST_DIR/$PKG_NAME"

# Output for GitHub Actions (if running in CI)
if [ -n "${GITHUB_OUTPUT:-}" ]; then
    echo "PKG_PATH=$DIST_DIR/$PKG_NAME" >> "$GITHUB_OUTPUT"
    echo "PKG_NAME=$PKG_NAME" >> "$GITHUB_OUTPUT"
    echo "VERSION=$VERSION" >> "$GITHUB_OUTPUT"
fi
