#!/bin/bash
#
# Installs fleetctl on macOS or Linux.
#
#   curl -sSL https://fleetdm.com/resources/install-fleetctl.sh | bash
#
# Optional environment variables:
#   FLEETCTL_VERSION      Version to install, e.g. 4.91.0 or v4.91.0. Defaults to the latest release.
#                         Can also be passed as the first argument: ... | bash -s -- 4.91.0
#   FLEETCTL_INSTALL_DIR  Directory to install the binary into. Defaults to $HOME/.fleetctl.

set -euo pipefail

FLEETCTL_INSTALL_DIR="${FLEETCTL_INSTALL_DIR:-${HOME}/.fleetctl}"
FLEETCTL_VERSION="${FLEETCTL_VERSION:-${1:-}}"
GITHUB_RELEASES="https://github.com/fleetdm/fleet/releases"

fail() {
  echo "Error: $*" >&2
  exit 1
}

for cmd in curl tar grep sed mktemp; do
  if ! command -v "$cmd" &> /dev/null; then
    fail "$cmd is not installed."
  fi
done

latest_version() {
  # github.com/.../releases/latest redirects to the newest non-prerelease tag and, unlike the
  # REST API, is not subject to the unauthenticated rate limit. Use the API only as a fallback.
  local tag
  tag="$(curl -sSfI "${GITHUB_RELEASES}/latest" 2>/dev/null | grep -i '^location:' | grep -o 'fleet-v[0-9][^[:space:]]*' || true)"
  if [[ -z "$tag" ]]; then
    tag="$(curl -sSfL "https://api.github.com/repos/fleetdm/fleet/releases/latest" 2>/dev/null | grep -o '"tag_name": *"fleet-v[^"]*"' | cut -d'"' -f4 || true)"
  fi
  echo "${tag#fleet-v}"
}

if [[ -z "$FLEETCTL_VERSION" || "$FLEETCTL_VERSION" == "latest" ]]; then
  echo "Fetching the latest version of fleetctl..."
  FLEETCTL_VERSION="$(latest_version)"
  [[ -n "$FLEETCTL_VERSION" ]] || fail "could not determine the latest fleetctl version."
fi
FLEETCTL_VERSION="${FLEETCTL_VERSION#v}"
if [[ ! "$FLEETCTL_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?$ ]]; then
  fail "invalid fleetctl version \"${FLEETCTL_VERSION}\" (expected e.g. 4.91.0)."
fi

if uname -m | grep -qE '^(arm|aarch64)'; then
  ARCH="arm64"
else
  ARCH="amd64"
fi

case "$(uname -s)" in
  Linux*)  OS="linux_${ARCH}" OS_DISPLAY_NAME="Linux" ;;
  Darwin*) OS="macos" OS_DISPLAY_NAME="macOS" ;;
  *)       fail "unsupported operating system: $(uname -s)" ;;
esac

ARCHIVE="fleetctl_v${FLEETCTL_VERSION}_${OS}"
DOWNLOAD_URL="${GITHUB_RELEASES}/download/fleet-v${FLEETCTL_VERSION}/${ARCHIVE}.tar.gz"
CHECKSUMS_URL="${GITHUB_RELEASES}/download/fleet-v${FLEETCTL_VERSION}/checksums.txt"

TMP_DIR="$(mktemp -d)"
STAGED=""
cleanup() {
  rm -rf "$TMP_DIR"
  [[ -n "$STAGED" ]] && rm -f "$STAGED"
  return 0
}
trap cleanup EXIT

echo "Downloading fleetctl ${FLEETCTL_VERSION} for ${OS_DISPLAY_NAME}..."
curl -sSfL "$DOWNLOAD_URL" -o "${TMP_DIR}/${ARCHIVE}.tar.gz" \
  || fail "could not download ${DOWNLOAD_URL}. Check that fleetctl ${FLEETCTL_VERSION} exists at ${GITHUB_RELEASES}."

# Verify the archive against the release's checksums when a SHA-256 tool is available.
if command -v sha256sum &> /dev/null; then
  SHA256="sha256sum"
elif command -v shasum &> /dev/null; then
  SHA256="shasum -a 256"
else
  SHA256=""
fi
if [[ -n "$SHA256" ]]; then
  EXPECTED="$(curl -sSfL "$CHECKSUMS_URL" | grep " ${ARCHIVE}.tar.gz\$" | cut -d' ' -f1 || true)"
  [[ -n "$EXPECTED" ]] || fail "could not find ${ARCHIVE}.tar.gz in ${CHECKSUMS_URL}."
  ACTUAL="$($SHA256 "${TMP_DIR}/${ARCHIVE}.tar.gz" | cut -d' ' -f1)"
  [[ "$EXPECTED" == "$ACTUAL" ]] || fail "checksum mismatch for ${ARCHIVE}.tar.gz (expected ${EXPECTED}, got ${ACTUAL})."
else
  echo "Warning: sha256sum or shasum not found, skipping checksum verification." >&2
fi

tar -xzf "${TMP_DIR}/${ARCHIVE}.tar.gz" -C "$TMP_DIR" --strip-components=1 "${ARCHIVE}/fleetctl"
[[ -f "${TMP_DIR}/fleetctl" ]] || fail "fleetctl binary not found in ${ARCHIVE}.tar.gz."

# Stage next to the destination (created exclusively by mktemp, so the name can't be guessed)
# and verify the new binary runs before the atomic rename replaces any existing fleetctl.
mkdir -p "$FLEETCTL_INSTALL_DIR"
STAGED="$(mktemp "${FLEETCTL_INSTALL_DIR}/.fleetctl.XXXXXX")"
cp "${TMP_DIR}/fleetctl" "$STAGED"
chmod 0755 "$STAGED"

STAGED_VERSION="$("$STAGED" --version 2>/dev/null | sed -n '1s/^fleetctl - version //p' || true)"
if [[ "$STAGED_VERSION" != "$FLEETCTL_VERSION" ]]; then
  fail "downloaded fleetctl does not run on this system or reports version \"${STAGED_VERSION}\" instead of ${FLEETCTL_VERSION}. Any existing fleetctl in ${FLEETCTL_INSTALL_DIR} was left unchanged."
fi
mv -f "$STAGED" "${FLEETCTL_INSTALL_DIR}/fleetctl"

echo "fleetctl ${FLEETCTL_VERSION} installed successfully in ${FLEETCTL_INSTALL_DIR}"
case ":${PATH}:" in
  *":${FLEETCTL_INSTALL_DIR}:"*|*":${FLEETCTL_INSTALL_DIR}/:"*) ;;
  *) echo "Add it to your PATH to run \"fleetctl\" directly: export PATH=\"${FLEETCTL_INSTALL_DIR}:\$PATH\"" ;;
esac
