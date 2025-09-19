#!/bin/sh

quit_app() {
  b="$1"
  if osascript -e "application id \"$b\" is running" >/dev/null 2>&1; then
    cu="$(stat -f "%Su" /dev/console 2>/dev/null || true)"
    if [ "$(id -u)" -eq 0 ] && [ "$cu" = "root" ]; then
      return 0
    fi
    i=0
    while [ "$i" -lt 10 ]; do
      osascript -e "tell application id \"$b\" to quit" >/dev/null 2>&1 || true
      if ! pgrep -f "$b" >/dev/null 2>&1; then
        break
      fi
      i=$((i+1))
      sleep 1
    done
  fi
}

[ -n "$INSTALLER_PATH" ] && [ -f "$INSTALLER_PATH" ] || { echo "missing installer"; exit 1; }

quit_app "com.logi.optionsplus"

TMPDIR="$(mktemp -d /tmp/logiopts_XXXXXX)" || exit 1
unzip -q "$INSTALLER_PATH" -d "$TMPDIR" || { echo "unzip failed"; rm -rf "$TMPDIR"; exit 1; }

# Try exact names first, then fallback search
APP_DIR=""
[ -d "$TMPDIR/logioptionsplus_installer.app" ] && APP_DIR="$TMPDIR/logioptionsplus_installer.app"
[ -z "$APP_DIR" ] && [ -d "$TMPDIR/Logi Options+ Installer.app" ] && APP_DIR="$TMPDIR/Logi Options+ Installer.app"
if [ -z "$APP_DIR" ]; then
  APP_DIR="$(/usr/bin/find "$TMPDIR" -type d \( \
    -iname "logioptionsplus_installer.app" -o \
    -iname "Logi Options+ Installer.app" -o \
    -iname "logi*installer*.app" -o \
    -iname "options+*.app" -o \
    -iname "install*.app" \
  \) -print -quit)"
fi

[ -n "$APP_DIR" ] && [ -d "$APP_DIR" ] || { echo "installer app not found"; rm -rf "$TMPDIR"; exit 1; }

# Binary path as in the cask
BIN="$APP_DIR/Contents/MacOS/logioptionsplus_installer"
if [ ! -x "$BIN" ]; then
  # Fallback: any executable in Contents/MacOS
  BIN="$(/usr/bin/find "$APP_DIR/Contents/MacOS" -maxdepth 1 -type f -perm -111 -print -quit 2>/dev/null)"
fi
[ -n "$BIN" ] && [ -x "$BIN" ] || { echo "installer binary not found"; rm -rf "$TMPDIR"; exit 1; }

# Run quietly
"$BIN" --quiet >/dev/null 2>&1 || { echo "installer app failed"; rm -rf "$TMPDIR"; exit 1; }

rm -rf "$TMPDIR" >/dev/null 2>&1 || true
echo "logi options+ installed"
