#!/bin/sh

# Best-effort uninstall: quit Battle.net, then remove the app bundle and
# common support directories. Errors are tolerated so partial state still
# results in a successful removal.

/usr/bin/pkill -x "Battle.net" 2>/dev/null || true
/usr/bin/pkill -x "Battle.net Helper" 2>/dev/null || true

/bin/rm -rf "/Applications/Battle.net.app" || true
/bin/rm -rf "/Applications/Battle.net" || true

for USER_HOME in /Users/*; do
  [ -d "$USER_HOME" ] || continue
  /bin/rm -rf "$USER_HOME/Library/Application Support/Battle.net" || true
  /bin/rm -rf "$USER_HOME/Library/Preferences/net.battle.app.plist" || true
  /bin/rm -rf "$USER_HOME/Library/Caches/net.battle.app" || true
done

exit 0
