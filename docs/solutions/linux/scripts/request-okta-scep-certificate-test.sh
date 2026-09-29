#!/bin/bash
# NOTE: This script deploys an Okta device certificate, but Okta Verify for Linux
# doesn't use it to mark the host as managed yet. Learn more:
# https://fleetdm.com/guides/deploy-okta-fastpass-for-linux
#
# Enroll this host's Okta device certificate over SCEP and install it.
#
# Uses scepclient v2.3.0 from micromdm/scep. If it isn't installed yet, the
# script downloads it (x86_64, 32-bit ARM) or builds it with Go (arm64).
#
# Optional overrides, useful for testing (defaults shown):
#   CERT_DIR=/etc/okta          where device.key and device.pem are installed
#   SCEP_CA_FINGERPRINT=        SHA-256 fingerprint (hex, colons optional) of the
#                               cert from GetCACert to encrypt the request to; the
#                               script logs each cert's fingerprint
#                               (default: let scepclient pick)
#   SCEPCLIENT_DIR=/usr/local/lib/okta-scep   where scepclient is installed
#   KEEP_WORK_DIR=0             1 = keep temp files for debugging (they contain
#                               the private key and challenge; delete afterwards)
set -eo pipefail
umask 077

CHALLENGE_URL="<Okta-challenge-URL>"
SCEP_URL="<Okta-SCEP-URL>"
SCEP_USERNAME="<Okta-SCEP-username>"
SCEP_PASSWORD="${FLEET_SECRET_OKTA_SCEP_PASSWORD}"
IDP_USERNAME="${FLEET_VAR_HOST_END_USER_IDP_USERNAME}"

CERT_DIR="${CERT_DIR:-/etc/okta}"
KEY_PATH="$CERT_DIR/device.key"
CERT_PATH="$CERT_DIR/device.pem"

log() { echo "[okta-scep] $*" >&2; }
die() { log "ERROR: $*"; exit 1; }

# --- Preflight -------------------------------------------------------------
for cmd in openssl curl; do
  command -v "$cmd" >/dev/null 2>&1 || die "'$cmd' is not installed"
done
case "$CHALLENGE_URL" in
  https://*) ;;
  *) die "CHALLENGE_URL must use HTTPS" ;;
esac
case "$SCEP_URL" in
  https://*) ;;
  *) die "SCEP_URL must use HTTPS" ;;
esac
case "$CHALLENGE_URL$SCEP_URL$SCEP_USERNAME" in
  *"<Okta-"*) die "Fill in CHALLENGE_URL, SCEP_URL and SCEP_USERNAME at the top of the script" ;;
esac
[ -n "$SCEP_PASSWORD" ] || die "FLEET_SECRET_OKTA_SCEP_PASSWORD is empty"
# IDP_USERNAME is optional; without it the CN is just "Okta FastPass".

# Private scratch dir; always cleaned up, including on failure.
WORK_DIR="$(mktemp -d /tmp/okta-scep.XXXXXX)"
cleanup() {
  rm -f "$KEY_PATH.new" "$CERT_PATH.new"
  if [ "${KEEP_WORK_DIR:-0}" = "1" ]; then
    log "Keeping $WORK_DIR (contains the private key and challenge; delete it when done)"
  else
    rm -rf "$WORK_DIR"
  fi
}
trap cleanup EXIT

# --- 0. scepclient ---------------------------------------------------------
# Needs micromdm scepclient v2.3.0 or later. Ubuntu's "scep" package ships
# v2.1.0, which can't parse Okta's response ("pkcs7: Message digest mismatch"),
# so the script installs its own pinned copy once and ignores any other one.
SCEPCLIENT_VERSION="v2.3.0"
SCEPCLIENT="${SCEPCLIENT_DIR:-/usr/local/lib/okta-scep}/scepclient-$SCEPCLIENT_VERSION"

apt_install() {
  command -v apt-get >/dev/null 2>&1 || die "Install $* first (no apt-get on this host)"
  log "Installing $* with apt-get"
  DEBIAN_FRONTEND=noninteractive apt-get install -y -q "$@" >/dev/null \
    || { apt-get update -q >/dev/null \
         && DEBIAN_FRONTEND=noninteractive apt-get install -y -q "$@" >/dev/null; } \
    || die "apt-get install $* failed"
}

install_scepclient() {
  local base="https://github.com/micromdm/scep" src="$WORK_DIR/scepclient" zip="" sha=""
  # SHA-256 of the v2.3.0 release files (GitHub doesn't publish checksums).
  case "$(uname -m)" in
    x86_64|amd64)  zip="scepclient-linux-amd64-$SCEPCLIENT_VERSION.zip"
                   sha="47b20e1b44b5789d4dd8572b17b22fe7d7d5fa6c1fe1b61f24fe26a3a1826a30" ;;
    armv7l|armv6l) zip="scepclient-linux-arm-$SCEPCLIENT_VERSION.zip"
                   sha="597272cb90080e2e03a4e42128d806ac940e00801ce340c3d3d4a938a57330d4" ;;
    aarch64|arm64) ;;
    *) die "No scepclient build for $(uname -m)" ;;
  esac

  if [ -n "$zip" ]; then
    log "Downloading scepclient $SCEPCLIENT_VERSION ($zip)"
    curl --fail --silent --show-error --location -o "$WORK_DIR/$zip" \
      "$base/releases/download/$SCEPCLIENT_VERSION/$zip"
    echo "$sha  $WORK_DIR/$zip" | sha256sum -c --quiet - || die "Checksum mismatch for $zip"
    command -v unzip >/dev/null 2>&1 || apt_install unzip
    unzip -q -o "$WORK_DIR/$zip" -d "$WORK_DIR/scepclient-zip"
    src="$WORK_DIR/scepclient-zip/${zip%-"$SCEPCLIENT_VERSION".zip}"
  else
    # The release has no arm64 binary, so build it from the tagged source.
    log "Building scepclient $SCEPCLIENT_VERSION from source (no arm64 release binary)"
    command -v go >/dev/null 2>&1 || apt_install golang-go
    curl --fail --silent --show-error --location -o "$WORK_DIR/scep.tar.gz" \
      "$base/archive/refs/tags/$SCEPCLIENT_VERSION.tar.gz"
    echo "d93239264aff09a0eb6097f0f5b828d9646a505be454320fca590283f2ca2482  $WORK_DIR/scep.tar.gz" \
      | sha256sum -c --quiet - || die "Checksum mismatch for the scepclient source"
    tar -xzf "$WORK_DIR/scep.tar.gz" -C "$WORK_DIR"
    ( cd "$WORK_DIR/scep-${SCEPCLIENT_VERSION#v}" \
      && HOME="$WORK_DIR" GOPATH="$WORK_DIR/go" GOCACHE="$WORK_DIR/gocache" GOFLAGS=-modcacherw \
         go build -o "$src" ./cmd/scepclient ) \
      || die "Building scepclient failed (check that the installed Go is recent enough)"
  fi

  install -d -m 755 "$(dirname "$SCEPCLIENT")"
  install -m 755 "$src" "$SCEPCLIENT"
  log "Installed $SCEPCLIENT"
}
[ -x "$SCEPCLIENT" ] || install_scepclient

# --- 1. Private key --------------------------------------------------------
# Built in WORK_DIR so a failed run never touches the currently installed key.
openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
  -out "$WORK_DIR/device.key" 2>/dev/null
# scepclient only reads PKCS#1 ("BEGIN RSA PRIVATE KEY") keys. Give it a PKCS#1
# copy of the same key; the installed device.key stays PKCS#8.
openssl pkey -in "$WORK_DIR/device.key" -traditional -out "$WORK_DIR/scep.key"

# --- 2. One-time challenge password ---------------------------------------
# Credentials are fed to curl on stdin so they don't show up in `ps`.
cred="$SCEP_USERNAME:$SCEP_PASSWORD"
cred="${cred//\\/\\\\}"
cred="${cred//\"/\\\"}"
printf 'user = "%s"\n' "$cred" | curl --config - "$CHALLENGE_URL" \
  --anyauth --fail --silent --show-error --location \
  -o "$WORK_DIR/challenge.html" \
  || die "Couldn't get a challenge from Okta. Check SCEP_USERNAME and the OKTA_SCEP_PASSWORD variable."
unset cred

# NDES returns UTF-16 HTML. Dropping NUL bytes decodes it and leaves the tags
# strippable, so this works whether the response is UTF-16 or UTF-8.
CHALLENGE_TEXT="$(tr -d '\0' < "$WORK_DIR/challenge.html" | sed 's/<[^>]*>/ /g' | tr -s '[:space:]' ' ')"
# Take the value right after "password is". Okta's challenges include - and _.
# No "first long token" fallback: on Okta's page that's the CA thumbprint.
CHALLENGE="$(printf '%s\n' "$CHALLENGE_TEXT" \
  | grep -oiE 'password is:? *[A-Za-z0-9_+/=-]{16,}' \
  | grep -oE '[A-Za-z0-9_+/=-]{16,}$' | head -n1 || true)"
[ -n "$CHALLENGE" ] || die "No challenge found in Okta's response (rerun with KEEP_WORK_DIR=1 and inspect challenge.html)"

# --- 3. CA chain -----------------------------------------------------------
# Logged for debugging and for choosing SCEP_CA_FINGERPRINT. scepclient fetches
# the chain again itself when it enrolls.
case "$SCEP_URL" in *\?*) sep='&' ;; *) sep='?' ;; esac
curl --fail --silent --show-error --location \
  "${SCEP_URL}${sep}operation=GetCACert" -o "$WORK_DIR/ca.der"
# A single CA comes back as a bare DER cert; a CA + RA chain as PKCS#7.
openssl x509 -inform DER -in "$WORK_DIR/ca.der" -out "$WORK_DIR/ca-chain.pem" 2>/dev/null \
  || openssl pkcs7 -inform DER -in "$WORK_DIR/ca.der" -print_certs -out "$WORK_DIR/ca-chain.pem" 2>/dev/null \
  || die "Could not parse the GetCACert response from $SCEP_URL"
awk -v dir="$WORK_DIR" '
  /BEGIN CERTIFICATE/ { f = dir "/ca-" (n++) ".pem"; p = 1 }
  p { print > f }
  /END CERTIFICATE/ { p = 0; close(f) }' "$WORK_DIR/ca-chain.pem"
for f in "$WORK_DIR"/ca-[0-9]*.pem; do
  [ -f "$f" ] || continue
  log "$(basename "$f"): $(openssl x509 -in "$f" -noout -subject) $(openssl x509 -in "$f" -noout -fingerprint -sha256)"
done

CA_FP=""
if [ -n "${SCEP_CA_FINGERPRINT:-}" ]; then
  CA_FP="$(printf '%s' "$SCEP_CA_FINGERPRINT" | tr -d ': ')"
  printf '%s' "$CA_FP" | grep -qiE '^[0-9a-f]{64}$' \
    || die "SCEP_CA_FINGERPRINT must be a SHA-256 fingerprint (64 hex digits)"
  log "Encrypting the request to the cert with fingerprint $CA_FP"
fi

# --- 4. Enroll -------------------------------------------------------------
# scepclient builds the CSR itself (CN + challengePassword) and runs GetCACert,
# GetCACaps and PKIOperation. Empty -organization/-ou/-country keep its
# defaults ("scep-client", "MDM", "US") out of the subject, so the CSR subject
# is just the CN.
# scepclient can only take the challenge as an argument, so it shows in `ps`
# while scepclient runs (it's one-time and short-lived).
scep_args=(
  -server-url "$SCEP_URL"
  -private-key "$WORK_DIR/scep.key"
  -certificate "$WORK_DIR/device.pem"
  -challenge "$CHALLENGE"
  -cn "${IDP_USERNAME:+$IDP_USERNAME }Okta FastPass"
  -organization "" -ou "" -country ""
)
[ -z "$CA_FP" ] || scep_args+=(-ca-fingerprint "$CA_FP")
"$SCEPCLIENT" "${scep_args[@]}" || die "scepclient enrollment failed"

# --- 5. Validate before installing ----------------------------------------
[ -s "$WORK_DIR/device.pem" ] || die "scepclient did not produce a certificate"
openssl x509 -in "$WORK_DIR/device.pem" -noout 2>/dev/null \
  || die "Issued certificate is not valid PEM"
cert_pub="$(openssl x509 -in "$WORK_DIR/device.pem" -noout -pubkey | openssl sha256)"
key_pub="$(openssl pkey -in "$WORK_DIR/device.key" -pubout | openssl sha256)"
[ "$cert_pub" = "$key_pub" ] || die "Issued certificate does not match the private key"

# --- 6. Install ------------------------------------------------------------
# Stage next to the targets, then rename, so the key/cert swap is near-atomic.
mkdir -p "$CERT_DIR"
install -m 600 "$WORK_DIR/device.key" "$KEY_PATH.new"
install -m 644 "$WORK_DIR/device.pem" "$CERT_PATH.new"
mv -f "$KEY_PATH.new" "$KEY_PATH"
mv -f "$CERT_PATH.new" "$CERT_PATH"

log "Installed $CERT_PATH: $(openssl x509 -in "$CERT_PATH" -noout -subject -enddate | tr '\n' ' ')"
