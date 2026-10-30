#!/bin/sh
# Removes AAA Certificate Services and emSign ECC Root CA - G3 from /etc/ssl/cert.pem, the bundle macOS's
# built-in curl, git, python3, perl, ruby, and openssl use without revocation checks
# (eMudhra intermediate CA compromise, https://emudhra.we-have-a-problem.com/#macos-curl).
# Apple's own TLS stack (Safari, URLSession) already blocks the compromised intermediates.
# Safe to run more than once. A macOS update can restore the original file.

BUNDLE="${1:-/etc/ssl/cert.pem}"

if [ ! -f "$BUNDLE" ]; then
  echo "No bundle at $BUNDLE, nothing to do."
  exit 0
fi

TMP="$BUNDLE.tmp.$$"
trap 'rm -f "$TMP"' EXIT

# cert.pem groups certificates under "### <Organization>" lines with one "=== <subject>" line per certificate.
awk '
  /^(###|===) / { skip = 0 }
  /^=== .*\/CN=(AAA Certificate Services|emSign ECC Root CA - G3)$/ { skip = 1 }
  !skip
' "$BUNDLE" > "$TMP" || exit 1

before=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$BUNDLE")
after=$(grep -c -- '-----BEGIN CERTIFICATE-----' "$TMP")
removed=$((before - after))

if [ "$removed" -eq 0 ]; then
  echo "$BUNDLE doesn't contain the affected roots ($before certificates)."
  exit 0
fi

if [ "$removed" -gt 2 ] || [ "$after" -lt 50 ]; then
  echo "Unexpected result ($before -> $after certificates), leaving $BUNDLE unchanged." >&2
  exit 1
fi

[ -f "$BUNDLE.bak-emudhra" ] || cp -p "$BUNDLE" "$BUNDLE.bak-emudhra" || exit 1
cat "$TMP" > "$BUNDLE" || exit 1

echo "Removed $removed certificate(s) from $BUNDLE ($before -> $after). Backup: $BUNDLE.bak-emudhra"
