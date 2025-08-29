#!/bin/sh

quit_application() {
  bundle_id="$1"
  timeout_duration=10
  if ! osascript -e "application id \"$bundle_id\" is running" >/dev/null 2>&1; then return; fi
  console_user="$(stat -f "%Su" /dev/console 2>/dev/null || true)"
  if [ "$(id -u)" -eq 0 ] && [ "$console_user" = "root" ]; then
    echo "Skipping quit for '$bundle_id'."
    return
  fi
  echo "Quitting '$bundle_id'..."
  i=0
  while [ "$i" -lt "$timeout_duration" ]; do
    osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1 || true
    if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
      echo "'$bundle_id' quit successfully."
      return
    fi
    i=$((i+1))
    sleep 1
  done
  echo "'$bundle_id' did not quit."
}

[ -n "$INSTALLER_PATH" ] && [ -f "$INSTALLER_PATH" ] || { echo "missing installer"; exit 1; }

quit_application "com.adobe.acc.AdobeCreativeCloud"

# Mount to a known path to avoid space-in-volume issues
MOUNT_POINT="$(mktemp -d "/tmp/adobe_cc.XXXXXX")"
if ! hdiutil attach -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" >/dev/null 2>&1; then
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
  echo "failed to mount dmg"
  exit 1
fi

# Find the installer app (handles Install.app or variants like *Install*.app)
INSTALL_APP="$(/usr/bin/find "$MOUNT_POINT" -maxdepth 5 -type d \( -name "Install.app" -o -iname "*Install*.app" \) -print -quit)"

if [ -z "$INSTALL_APP" ] || [ ! -d "$INSTALL_APP" ]; then
  hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
  rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true
  echo "Install.app not found"
  exit 1
fi

# Prefer Contents/MacOS/Install, fall back to first executable in MacOS
BIN="$INSTALL_APP/Contents/MacOS/Install"
if [ ! -x "$BIN" ]; then
  BIN="$(/usr/bin/find "$INSTALL_APP/Contents/MacOS" -type f -perm +111 -print -quit 2>/dev/null)"
fi
[ -n "$BIN" ] && [ -x "$BIN" ] || { hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true; rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true; echo "installer binary not found"; exit 1; }

# Try silent, fall back to normal if needed
"$BIN" --mode=silent >/dev/null 2>&1 || "$BIN" >/dev/null 2>&1

hdiutil detach "$MOUNT_POINT" >/dev/null 2>&1 || true
rmdir "$MOUNT_POINT" >/dev/null 2>&1 || true

echo "adobe creative cloud installed"
