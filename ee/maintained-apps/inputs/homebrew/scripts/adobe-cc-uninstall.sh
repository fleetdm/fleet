#!/bin/sh
# Verbose logging to a per-run logfile (noisy output goes here, not to stdout)
LOGFILE="/var/tmp/appcmd-uninstall-$$.log"
set -x
exec 1>"$LOGFILE" 2>&1
echo "[$(date -u +%FT%TZ)] start, logfile=$LOGFILE"

BUNDLE_ID="com.adobe.acc.AdobeCreativeCloud"

# --- Quit Creative Cloud politely in the console user's GUI session, then hard-stop stragglers ---
console_user="$(stat -f '%Su' /dev/console 2>/dev/null || echo root)"
console_uid="$(id -u "$console_user" 2>/dev/null || echo 0)"

is_running_gui() {
  launchctl asuser "$console_uid" osascript -e "application id \"$BUNDLE_ID\" is running" >/dev/null 2>&1
}

quit_app() {
  # Friendly quit via AppleScript (if running in the GUI session)
  if is_running_gui; then
    i=0
    while [ $i -lt 20 ]; do
      launchctl asuser "$console_uid" osascript -e "tell application id \"$BUNDLE_ID\" to quit" || true
      sleep 1
      is_running_gui || break
      i=$((i+1))
    done
  fi

  # Hard stop common ACC background processes that keep the uninstaller blocked
  pkill -f "/Creative Cloud\.app/Contents/MacOS/Creative Cloud" || true
  pkill -f "[A]dobe Desktop Service" || true
  pkill -f "[C]ore Sync" || true
  pkill -f "[C]CLibrary" || true
  pkill -f "[C]CXProcess" || true
}

quit_app

# --- Locate Adobe's official uninstaller app and its binary ---
UNINST_APP=""
for p in \
  "/Applications/Utilities/Adobe Creative Cloud/Utils/Creative Cloud Uninstaller.app" \
  "/Applications/Adobe Creative Cloud/Utils/Creative Cloud Uninstaller.app" \
  "/Applications/Utilities/Adobe Creative Cloud/Creative Cloud Uninstaller.app"
do
  [ -d "$p" ] && { UNINST_APP="$p"; break; }
done

# Fallback: search (no -maxdepth for BSD find compatibility)
[ -z "$UNINST_APP" ] && UNINST_APP="$(/usr/bin/find /Applications /Applications/Utilities -type d -iname '*creative*cloud*uninstaller*.app' -print -quit 2>/dev/null || true)"

BIN=""
if [ -n "$UNINST_APP" ] && [ -d "$UNINST_APP" ]; then
  BIN="$(/usr/bin/find "$UNINST_APP/Contents/MacOS" -type f -perm -111 -print -quit 2>/dev/null || true)"
fi

# --- Run the uninstaller with a short watchdog so it can't hang forever ---
run_with_timeout() { # usage: run_with_timeout <secs> <cmd...>
  python3 - "$@" <<'PY'
import os, signal, subprocess, sys, time
secs=int(sys.argv[1]); cmd=sys.argv[2:]
p=subprocess.Popen(cmd, preexec_fn=os.setsid)
deadline=time.time()+secs
while time.time()<deadline:
    rc=p.poll()
    if rc is not None: sys.exit(rc)
    time.sleep(0.5)
# timeout -> kill the whole process group
try:
    os.killpg(p.pid, signal.SIGKILL)
except Exception:
    pass
sys.exit(124)
PY
}

if [ -n "$BIN" ]; then
  # Clear quarantine in case Gatekeeper would kill it
  xattr -dr com.apple.quarantine "$UNINST_APP" 2>/dev/null || true
  echo "[$(date -u +%FT%TZ)] running adobe uninstaller: $BIN -uninstall --force"
  run_with_timeout 180 "$BIN" -uninstall --force || {
    echo "[$(date -u +%FT%TZ)] uninstaller timed out or failed; proceeding to manual cleanup"
  }
fi

# --- Remove app bundles ---
rm -rf "/Applications/Adobe Creative Cloud.app" \
       "/Applications/Creative Cloud.app" \
       "/Applications/Utilities/Adobe Creative Cloud/ACC/Creative Cloud.app" || true

# --- System support files ---
rm -rf "/Library/Application Support/Adobe/Creative Cloud" \
       "/Library/Application Support/Adobe/Adobe Desktop Common" || true
rm -f  "/Library/Preferences/com.adobe.acc.AdobeCreativeCloud.plist" || true
rm -f  /var/db/receipts/com.adobe.acc*.bom /var/db/receipts/com.adobe.acc*.plist || true

# --- Per-user cleanup ---
for udir in /Users/* /var/root; do
  [ -d "$udir/Library" ] || continue
  rm -rf "$udir/Library/Application Support/Adobe/Creative Cloud" \
         "$udir/Library/Caches/com.adobe.acc.AdobeCreativeCloud" \
         "$udir/Library/Logs/Adobe/Creative Cloud" || true
  rm -f  "$udir/Library/Preferences/com.adobe.acc.AdobeCreativeCloud.plist" || true
done

echo "[$(date -u +%FT%TZ)] adobe creative cloud uninstalled (logfile: $LOGFILE)"