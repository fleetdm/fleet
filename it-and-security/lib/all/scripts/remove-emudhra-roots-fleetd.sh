#!/bin/sh
# Removes the emSign roots excluded by curl's CA bundle (eMudhra intermediate CA compromise,
# https://emudhra.we-have-a-problem.com) and the legacy Comodo AAA root from fleetd's osquery CA bundle.
# Safe to run more than once.

BUNDLE="${1:-/opt/orbit/certs.pem}"

if [ ! -f "$BUNDLE" ]; then
  echo "No bundle at $BUNDLE, nothing to do."
  exit 0
fi

TMP="$BUNDLE.tmp.$$"
trap 'rm -f "$TMP"' EXIT

# certs.pem uses mk-ca-bundle's layout: a name line, an "=====" underline, then the PEM block.
awk '
  BEGIN {
    drop["emSign ECC Root CA - G3"] = 1
    drop["emSign Root CA - C1"] = 1
    drop["emSign ECC Root CA - C3"] = 1
    drop["Comodo AAA Services root"] = 1
  }
  { line[NR] = $0 }
  END {
    skip = 0
    for (i = 1; i <= NR; i++) {
      if (!skip && (line[i] in drop) && line[i + 1] ~ /^=+$/) skip = 1
      if (!skip) print line[i]
      if (skip && line[i] ~ /^-----END CERTIFICATE-----/) skip = 0
    }
  }
' "$BUNDLE" > "$TMP" || exit 1

before=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$BUNDLE")
after=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$TMP")
removed=$((before - after))

if [ "$removed" -eq 0 ]; then
  echo "$BUNDLE doesn't contain the affected roots ($before certificates)."
  exit 0
fi

# Guard against a malformed bundle stripping more than the four named roots.
if [ "$removed" -gt 4 ] || [ "$after" -lt 50 ]; then
  echo "Unexpected result ($before -> $after certificates), leaving $BUNDLE unchanged." >&2
  exit 1
fi

[ -f "$BUNDLE.bak-emudhra" ] || cp -p "$BUNDLE" "$BUNDLE.bak-emudhra" || exit 1
# Overwrite in place to keep the file's owner and mode.
cat "$TMP" > "$BUNDLE" || exit 1

echo "Removed $removed certificate(s) from $BUNDLE ($before -> $after). Backup: $BUNDLE.bak-emudhra"
