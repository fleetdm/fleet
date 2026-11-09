#!/bin/sh
# Installs the Inter font (Google Fonts build, upright + italic variable fonts)
# for all users on macOS (/Library/Fonts) and Linux (/usr/local/share/fonts).
# Downloads are pinned to a google/fonts commit and verified by SHA-256.

set -eu

BASE_URL="https://raw.githubusercontent.com/google/fonts/e1d6480102fed30739fead0faee463101f892c8f/ofl/inter"

case "$(uname -s)" in
  Darwin) DEST="/Library/Fonts" ;;
  Linux) DEST="/usr/local/share/fonts/inter" ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -d ' ' -f 1
  else
    shasum -a 256 "$1" | cut -d ' ' -f 1
  fi
}

download() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 -o "$2" "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$2" "$1"
  else
    echo "curl or wget is required" >&2
    exit 1
  fi
}

mkdir -p "$DEST"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
changed=0

install_font() {
  remote="$1" file="$2" expected="$3"
  target="$DEST/$file"
  if [ -f "$target" ] && [ "$(sha256 "$target")" = "$expected" ]; then
    echo "$target is already installed"
    return
  fi
  download "$BASE_URL/$remote" "$TMP/$file"
  actual="$(sha256 "$TMP/$file")"
  if [ "$actual" != "$expected" ]; then
    echo "Checksum mismatch for $file: expected $expected, got $actual" >&2
    exit 1
  fi
  chmod 644 "$TMP/$file"
  mv -f "$TMP/$file" "$target"
  echo "Installed $target"
  changed=1
}

install_font "Inter%5Bopsz,wght%5D.ttf" "Inter-VariableFont_opsz,wght.ttf" \
  29160a80ff49ddcab2c97711247e08b1fab27a484a329ce8b813d820dc559031
install_font "Inter-Italic%5Bopsz,wght%5D.ttf" "Inter-Italic-VariableFont_opsz,wght.ttf" \
  acd98e64795781b2058f07b18475e0ecee2a0fe2b42a49e2f9e37d0d6bf66ce6

if [ "$changed" -eq 1 ] && [ "$DEST" != "/Library/Fonts" ] && command -v fc-cache >/dev/null 2>&1; then
  fc-cache -f "$DEST"
fi
