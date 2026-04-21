#!/bin/bash
# Enables Touch ID authentication for sudo on macOS 15+.

set -euo pipefail

PAM_TID_LINE="auth       sufficient     pam_tid.so"
SUDO_LOCAL="/etc/pam.d/sudo_local"
SUDO_LOCAL_TEMPLATE="/etc/pam.d/sudo_local.template"

log() {
  echo "[enable-touch-id-sudo] $1"
}

if [[ $EUID -ne 0 ]]; then
  log "This script must run as root. Fleet scripts run as root by default."
  exit 1
fi

if [[ ! -f "$SUDO_LOCAL_TEMPLATE" ]]; then
  log "Expected $SUDO_LOCAL_TEMPLATE on macOS 15+, but it was not found."
  exit 1
fi

if [[ ! -f "$SUDO_LOCAL" ]]; then
  log "Creating $SUDO_LOCAL from template."
  cp "$SUDO_LOCAL_TEMPLATE" "$SUDO_LOCAL"
fi

if grep -Eq '^[[:space:]]*auth[[:space:]]+sufficient[[:space:]]+pam_tid\.so' "$SUDO_LOCAL"; then
  log "Touch ID for sudo is already enabled in $SUDO_LOCAL."
  exit 0
fi

if grep -Eq '^[[:space:]]*#[[:space:]]*auth[[:space:]]+sufficient[[:space:]]+pam_tid\.so' "$SUDO_LOCAL"; then
  log "Uncommenting pam_tid line in $SUDO_LOCAL."
  # Portable in-place edit (macOS sed requires the empty "" backup arg).
  sed -i "" -E 's/^[[:space:]]*#[[:space:]]*(auth[[:space:]]+sufficient[[:space:]]+pam_tid\.so)/\1/' "$SUDO_LOCAL"
else
  log "Appending pam_tid line to $SUDO_LOCAL."
  printf '%s\n' "$PAM_TID_LINE" >> "$SUDO_LOCAL"
fi

log "Touch ID for sudo enabled via $SUDO_LOCAL."
