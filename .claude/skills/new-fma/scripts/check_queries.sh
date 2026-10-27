#!/bin/bash
# Run a macOS FMA output's exists/patched/open queries through real osquery on this
# Mac, plus patched with the version bumped past anything real (must return 0 rows
# when the app is installed, or the policy can never flag an outdated host).
# Usage: check_queries.sh ee/maintained-apps/outputs/<app>/darwin.json
set -euo pipefail
manifest=$1
OSQ=${OSQ:-/opt/orbit/bin/osqueryd/macos-app/stable/osquery.app/Contents/MacOS/osqueryd}
[ -x "$OSQ" ] || { echo "osqueryd not found at $OSQ; set OSQ (see references/queries.md)" >&2; exit 2; }

rows() { "$OSQ" -S --json "$1" 2>/dev/null | jq length; }

version=$(jq -r '.versions[0].version' "$manifest")
for q in exists patched open; do
  sql=$(jq -r ".versions[0].queries.$q // empty" "$manifest")
  if [ -z "$sql" ]; then printf '%-22s (no query)\n' "$q"; continue; fi
  printf '%-22s %s rows\n' "$q" "$(rows "$sql")"
done

major=${version%%[!0-9]*}
if [ -n "$major" ]; then
  bumped="$((major + 1000))${version#"$major"}"
  sql=$(jq -r '.versions[0].queries.patched' "$manifest")
  sql=${sql//"'$version'"/"'$bumped'"}
  printf '%-22s %s rows\n' "patched@$bumped" "$(rows "$sql")"
fi
cat <<'TXT'
Expected with the app installed and closed: exists >=1, patched 1 (0 if this host
really is behind), open 1, patched@bumped 0. Launch the app and re-run: open 0.
TXT
