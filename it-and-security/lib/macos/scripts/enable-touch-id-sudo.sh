#!/bin/bash
# Enables Touch ID authentication for sudo on macOS 15+.

set -euo pipefail

if test "$(id -u)" -ne 0; then
  echo "[enable-touch-id-sudo] This script must run as root. Fleet scripts run as root by default."
  exit 1
fi

if test ! -f /etc/pam.d/sudo_local.template; then
  echo "[enable-touch-id-sudo] Expected /etc/pam.d/sudo_local.template on macOS 15+, but it was not found."
  exit 1
fi

if test ! -f /etc/pam.d/sudo_local; then
  echo "[enable-touch-id-sudo] Creating /etc/pam.d/sudo_local from template."
  cp /etc/pam.d/sudo_local.template /etc/pam.d/sudo_local
fi

if grep -Eq '^[[:space:]]*auth[[:space:]]+sufficient[[:space:]]+pam_tid\.so' /etc/pam.d/sudo_local; then
  echo "[enable-touch-id-sudo] Touch ID for sudo is already enabled in /etc/pam.d/sudo_local."
  exit 0
fi

if grep -Eq '^[[:space:]]*#[[:space:]]*auth[[:space:]]+sufficient[[:space:]]+pam_tid\.so' /etc/pam.d/sudo_local; then
  echo "[enable-touch-id-sudo] Uncommenting pam_tid line in /etc/pam.d/sudo_local."
  sed -i "" -E 's/^[[:space:]]*#[[:space:]]*(auth[[:space:]]+sufficient[[:space:]]+pam_tid\.so)/\1/' /etc/pam.d/sudo_local
else
  echo "[enable-touch-id-sudo] Appending pam_tid line to /etc/pam.d/sudo_local."
  printf '%s\n' 'auth       sufficient     pam_tid.so' >> /etc/pam.d/sudo_local
fi

echo "[enable-touch-id-sudo] Touch ID for sudo enabled via /etc/pam.d/sudo_local."
