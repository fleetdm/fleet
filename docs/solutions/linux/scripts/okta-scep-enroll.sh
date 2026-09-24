#!/bin/bash
# Enroll this host's Okta device certificate over SCEP and install it.
#
# Uses scepclient from micromdm/scep (Ubuntu/Debian package "scep").
#
# Optional overrides, useful for testing (defaults shown):
#   CERT_DIR=/etc/okta          where device.key and device.pem are installed
#   SCEP_CA_FINGERPRINT=        SHA-256 fingerprint (hex, colons optional) of the
#                               cert from GetCACert to encrypt the request to; the
#                               script logs each cert's fingerprint
#                               (default: let scepclient pick)
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
for cmd in openssl curl scepclient; do
  command -v "$cmd" >/dev/null 2>&1 || die "'$cmd' is not installed"
done
case "$CHALLENGE_URL$SCEP_URL$SCEP_USERNAME" in
  *"<Okta-"*) die "Fill in CHALLENGE_URL, SCEP_URL and SCEP_USERNAME at the top of the script" ;;
esac
[ -n "$SCEP_PASSWORD" ] || die "FLEET_SECRET_OKTA_SCEP_PASSWORD is empty"
[ -n "$IDP_USERNAME" ]  || die "FLEET_VAR_HOST_END_USER_IDP_USERNAME is empty"

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
  -o "$WORK_DIR/challenge.html"
unset cred

# NDES returns UTF-16 HTML. Dropping NUL bytes decodes it and leaves the tags
# strippable, so this works whether the response is UTF-16 or UTF-8.
CHALLENGE_TEXT="$(tr -d '\0' < "$WORK_DIR/challenge.html" | sed 's/<[^>]*>/ /g' | tr -s '[:space:]' ' ')"
# Prefer the value right after "password is"; fall back to the first long token.
CHALLENGE="$(printf '%s\n' "$CHALLENGE_TEXT" \
  | grep -oiE 'password is:? *[A-Za-z0-9]{16,}' \
  | grep -oE '[A-Za-z0-9]{16,}$' | head -n1 || true)"
if [ -z "$CHALLENGE" ]; then
  CHALLENGE="$(printf '%s\n' "$CHALLENGE_TEXT" | grep -oE '[A-Za-z0-9]{16,}' | head -n1 || true)"
fi
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
# Caveats of scepclient 2.1.0, with no flags to change them:
#   - the challenge can only be passed as an argument, so it shows in `ps`
#     while scepclient runs (it's one-time and short-lived)
#   - the SCEP message uses DES-CBC encryption and SHA-1 signing; TLS on
#     SCEP_URL still protects it in transit
scep_args=(
  -server-url "$SCEP_URL"
  -private-key "$WORK_DIR/scep.key"
  -certificate "$WORK_DIR/device.pem"
  -challenge "$CHALLENGE"
  -cn "$IDP_USERNAME Okta FastPass"
  -organization "" -ou "" -country ""
)
[ -z "$CA_FP" ] || scep_args+=(-ca-fingerprint "$CA_FP")
scepclient "${scep_args[@]}" || die "scepclient enrollment failed"

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
