# Deploy Okta FastPass for Linux

Okta's FastPass can require a managed device, confirmed by a certificate from MDM, in addition to a verified user. This guide installs Okta Verify and deploys that device certificate with Fleet, so Linux hosts meet the same managed-device requirement as macOS and Windows. 

See Okta's [Okta Verify for Linux release notes](https://help.okta.com/oie/en-us/content/topics/releasenotes/ov/ov-release-notes-linux.htm) for supported Linux distributions and what's new in each release.

> **Note:** You can deploy and configure Okta FastPass for Linux today (Steps 1 and 2) if you request access from Okta. Marking Linux hosts as managed (Steps 3 through 5) is in the works and needs:
> - **Fleet:** Okta certificate authority (CA) support in Fleet's "Request certificate" API endpoint, tracked in [fleetdm/fleet#52993](https://github.com/fleetdm/fleet/issues/52993). The endpoint currently only supports Hydrant and custom EST CAs, so the Step 4 script fails until #52993 ships. To test before then, use the script in [Test with Okta's SCEP endpoint directly](#test-with-oktas-scep-endpoint-directly).
> - **Okta:** Linux support in Okta's device management platform integration. Today, Okta's **Add platform** option covers Windows and macOS only.

## Step 1: Download Okta Verify for Linux

1. Contact Okta to request access to Okta Verify for Linux.
2. In Okta, head to **Settings > Downloads**.
3. Under **Desktop Apps**, find **Okta Verify for Linux** and select **Download latest**.

## Step 2: Install Okta Verify on Linux hosts at enrollment

1. In Fleet, head to **Software**, choose a fleet, and select **Add software > Custom package** to upload the Okta Verify `.deb` file you downloaded in Step 1.
2. Head to **Controls > Setup experience > Install software**, select the **Linux** tab, and check **Okta Verify** so it installs automatically on every Linux host as it enrolls.

## Step 3: Connect Fleet to Okta's CA

> **Note:** Steps 3 through 5 mark Linux hosts as managed. This is in the works and depends on [fleetdm/fleet#52993](https://github.com/fleetdm/fleet/issues/52993) and Linux support in Okta's device management platform integration. Okta's **Add platform** option below currently covers Windows and macOS only.

1. In Okta, head to **Security > Device integrations**, select **Add platform**, then choose **Desktop (Windows and macOS only)**.
2. On the **Add device management platform** page, select **Use Okta as Certificate Authority** and **Dynamic SCEP URL** (verify **Generic** is selected), then select **Generate**.
3. Copy the **Password** (you'll use it as the Challenge in the next step), then select **Save**.
4. In Fleet, head to **Settings > Integrations > Certificate authorities**.
5. Select **Add certificate authority**, then choose **Dynamic SCEP - Okta CA or Microsoft Network Device Enrollment Service (NDES)** in the dropdown. Okta uses NDES under the hood.
6. Enter the **SCEP URL** from Okta and the **Challenge** password you copied in step 3.
7. Select **Add CA**. Your Okta CA now appears in Fleet's list of certificate authorities. Note its `id`. You'll need it in Step 4 (find it via [`GET /certificate_authorities`](https://fleetdm.com/docs/rest-api/rest-api#list-certificate-authorities-cas)).

## Step 4: Deploy the FastPass certificate with a script-only package

> **Note:** This step depends on [fleetdm/fleet#52993](https://github.com/fleetdm/fleet/issues/52993), which adds Okta CA support to Fleet's "Request certificate" API endpoint. That endpoint currently only supports Hydrant and custom EST CAs, so the script below fails until #52993 ships. To test before then, use the script in [Test with Okta's SCEP endpoint directly](#test-with-oktas-scep-endpoint-directly).

Okta Verify needs a device certificate from your Okta CA to unlock FastPass. Deploy it with a script-only software package that requests the certificate from Fleet's ["Request certificate" API endpoint](https://fleetdm.com/docs/rest-api/rest-api#request-certificate) and writes it where Okta Verify expects it, so it runs alongside Okta Verify during setup experience.

1. Create an API-only user with the global maintainer role. Learn how in the [API-only user guide](https://fleetdm.com/guides/fleetctl#create-api-only-user). For least privilege, restrict the user to only the [Request certificate](https://fleetdm.com/docs/rest-api/rest-api#request-certificate) endpoint by passing its `id` in `api_endpoints` when you create the user. Find the `id` with [`GET /rest_api`](https://fleetdm.com/docs/rest-api/rest-api#list-api-endpoints-for-api-only-user-permissions).
2. In Fleet, head to **Controls > Variables** and create a variable called `REQUEST_CERTIFICATE_API_TOKEN` with the API-only user's API token as its value. The script below reads it as `$FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN`.
3. In your text editor, copy the script below, then replace `<Fleet-server-URL>` and `<Okta-CA-ID>` (the CA `id` from Step 3) with your own values.


```shell
#!/bin/bash
set -e

FLEET_URL="<Fleet-server-URL>"
CA_ID="<Okta-CA-ID>"
CERT_DIR="/etc/okta"
KEY_PATH="$CERT_DIR/device.key"
CERT_PATH="$CERT_DIR/device.pem"

mkdir -p "$CERT_DIR"

# Generate a private key and CSR identifying this host's device certificate.
openssl genpkey -algorithm RSA -out "$KEY_PATH" -pkeyopt rsa_keygen_bits:2048
chmod 600 "$KEY_PATH"

openssl req -new -sha256 -key "$KEY_PATH" -out /tmp/okta-verify.csr -subj "/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME Okta FastPass"

# Escape the CSR for the JSON request body.
CSR=$(sed 's/$/\\n/' /tmp/okta-verify.csr | tr -d '\n')
REQUEST='{ "csr": "'"${CSR}"'", "return_pem_certificate": true }'

curl "${FLEET_URL}/api/latest/fleet/certificate_authorities/${CA_ID}/request_certificate" \
  -X 'POST' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'authorization: Bearer '"$FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN" \
  -H 'content-type: application/json' \
  --data-raw "${REQUEST}" -o /tmp/okta-verify-response.json

jq -r .certificate /tmp/okta-verify-response.json > "$CERT_PATH"
chmod 644 "$CERT_PATH"

rm -f /tmp/okta-verify.csr /tmp/okta-verify-response.json
```

By default, the `certificate` field in the response is a PEM-encoded PKCS7 envelope, not a standard x509 certificate. The script above passes `"return_pem_certificate": true` so Fleet returns a `-----BEGIN CERTIFICATE-----` block that can be written directly to `device.pem`.


4. In Fleet, head to **Software**, select **Add software > Custom package**, and upload the script above as a `.sh` file (a script with no installer becomes a [script-only package](https://fleetdm.com/guides/deploy-software-packages#script-only-packages)).
5. Head to **Controls > Setup experience > Install software**, select the **Linux** tab, and check the new script-only package so it runs automatically alongside Okta Verify during enrollment.

### Test with Okta's SCEP endpoint directly

Until [fleetdm/fleet#52993](https://github.com/fleetdm/fleet/issues/52993) ships, use this script to test certificate issuance. It skips Fleet's "Request certificate" API endpoint and requests the certificate straight from Okta's SCEP endpoint. It's for testing only, because the Okta SCEP password is available to every host that runs it.

1. Install the `scep` package on the test host (`sudo apt install scep`). It provides [`scepclient`](https://github.com/micromdm/scep), which the script uses to enroll with Okta's SCEP endpoint.
2. In Fleet, head to **Controls > Variables** and create a variable called `OKTA_SCEP_PASSWORD` with the password from Okta's **Add device management platform** page in Step 3. The script reads it as `$FLEET_SECRET_OKTA_SCEP_PASSWORD`.
3. In your text editor, copy the script below, then replace `<Okta-challenge-URL>`, `<Okta-SCEP-URL>`, and `<Okta-SCEP-username>` with the values from the same Okta page.
4. In Fleet, head to **Controls > Scripts**, upload the script, then run it on the test host from **Host details > Actions > Run script**.

```shell
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
```

## Step 5: Renew or restore the certificate automatically

> **Note:** This step renews the certificate from Step 4, so it also depends on [fleetdm/fleet#52993](https://github.com/fleetdm/fleet/issues/52993).

Okta Verify for Linux isn't covered by Fleet's [automatic certificate renewal](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#renewal). The script-only package in Step 4 only installs once, during setup experience, so it won't fix a certificate that's later deleted or expires. Wire that same package to a policy, so Fleet reinstalls it, and renews the certificate, whenever a host fails the check.

1. In Fleet, head to **Policies** and select **Add policy**. Use the following query to detect whether the certificate is missing or expires in the next 30 days:

```sql
SELECT 1 FROM certificates WHERE path = '/etc/okta/device.pem' AND not_valid_after > (CAST(strftime('%s', 'now') AS INTEGER) + 2592000);
```

2. Select **Save**, target only **Linux**, then select **Save** again.
3. On the **Policies** page, select **Manage automations**, then select **Install software**.
4. Select your new policy, then in the dropdown, choose the script-only package you uploaded in Step 4.
5. Now, any Linux host missing `/etc/okta/device.pem`, or whose certificate expires within 30 days, fails the policy, and Fleet reinstalls the package to renew it.

## Verify

1. Enroll a Linux host (or wait for an existing one to check in after these changes).
2. In Okta, head to **Directory > Devices** and confirm the host appears with **Platform** Linux and **Enrolled By** Okta Verify.
3. On the host, open Okta Verify and confirm FastPass is available for sign-in.
4. After Steps 3 through 5 are available, confirm `/etc/okta/device.pem` exists on the host and is a valid certificate: `openssl x509 -in /etc/okta/device.pem -noout -text` (or `openssl pkcs7 -print_certs -in /etc/okta/device.pem` if Fleet returned a PKCS7 envelope).

<meta name="articleTitle" value="Deploy Okta FastPass for Linux">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-14">
<meta name="description" value="Deploy Okta Verify and its FastPass device certificate to Linux hosts with Fleet.">
