# Configure Okta FastPass for Linux

_Available in Fleet Premium_

This guide uses Fleet, but works with any MDM or tool that can deploy apps and run scripts on Linux: install the Okta Verify client and issue the device certificate Okta Verify uses for FastPass, so end users on Linux hosts can authenticate the same way they already do on macOS and Windows.

This guide shows how to:
1. Connect Fleet to Okta's certificate authority (CA).
2. Install Okta Verify on Linux hosts at enrollment.
3. Deploy the Okta Verify FastPass certificate with a script-only software package.

## Prerequisites

- Fleet Premium, with Linux hosts enrolled.
- Admin access in both Okta and Fleet.
- `fleetd` deployed with [scripts enabled](https://fleetdm.com/guides/scripts#enable-scripts).
- The host has `openssl`, `sed`, `curl`, and `jq` installed.

## Step 1: Download Okta Verify for Linux

1. In Okta, head to **Settings > Downloads**.
2. Under **Desktop Apps**, find **Okta Verify for Linux** and select **Download latest**.

## Step 2: Connect Fleet to Okta's CA

1. In Okta, head to **Security > Device integrations**, select **Add platform**, then choose **Desktop (Windows and macOS only)**.
2. On the **Add device management platform** page, select **Use Okta as Certificate Authority** and **Dynamic SCEP URL** (verify **Generic** is selected), then select **Generate**.
3. Copy the **Password** (you'll use it as the Challenge in the next step), then select **Save**.
4. In Fleet, head to **Settings > Integrations > Certificate authorities**.
5. Select **Add certificate authority**, then choose **Dynamic SCEP - Okta CA or Microsoft Network Device Enrollment Service (NDES)** in the dropdown. Okta uses NDES under the hood.
6. Enter the **SCEP URL** from Okta and the **Challenge** password you copied in step 3.
7. Select **Add CA**. Your Okta CA now appears in Fleet's list of certificate authorities. Note its `id` — you'll need it in Step 4 (find it via [`GET /certificate_authorities`](https://fleetdm.com/docs/rest-api/rest-api#list-certificate-authorities-cas)).

## Step 3: Install Okta Verify on Linux hosts at enrollment

1. In Fleet, head to **Software**, choose the team, and select **Add software > Custom package** to upload the Okta Verify `.deb` file you downloaded in Step 1.
2. Head to **Controls > Setup experience > Install software**, select the **Linux** tab, and check **Okta Verify** so it installs automatically on every Linux host as it enrolls.

## Step 4: Deploy the FastPass certificate with a script-only package

Okta Verify needs a device certificate from your Okta CA to unlock FastPass. Deploy it with a script-only software package that requests the certificate from Fleet's ["Request certificate" API endpoint](https://fleetdm.com/docs/rest-api/rest-api#request-certificate) and writes it where Okta Verify expects it, so it runs alongside Okta Verify during setup experience.

1. Create an API-only user with the global maintainer role. Learn how in the [API-only user guide](https://fleetdm.com/guides/fleetctl#create-api-only-user).
2. In Fleet, head to **Controls > Variables** and create a variable called `REQUEST_CERTIFICATE_API_TOKEN` with the API-only user's API token as its value. The script below reads it as `$FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN`.
3. In your text editor, copy the script below, then replace `<Fleet-server-URL>` and `<Okta-CA-ID>` (the CA `id` from Step 2) with your own values.


```shell
#!/bin/bash
set -e

FLEET_URL="<Fleet-server-URL>"
CA_ID="<Okta-CA-ID>"
CERT_DIR="/opt/okta-verify"
KEY_PATH="$CERT_DIR/device.key"
CERT_PATH="$CERT_DIR/device.pem"

mkdir -p "$CERT_DIR"
chmod 700 "$CERT_DIR"

# Generate a private key and CSR identifying this host's device certificate.
openssl genpkey -algorithm RSA -out "$KEY_PATH" -pkeyopt rsa_keygen_bits:2048
chmod 600 "$KEY_PATH"

host_identifier="$(hostnamectl --static 2>/dev/null || hostname)"
openssl req -new -sha256 -key "$KEY_PATH" -out /tmp/okta-verify.csr -subj "/CN=${host_identifier}"

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

## Verify

1. Enroll a Linux host (or wait for an existing one to check in after these changes).
2. On the host, confirm `/opt/okta-verify/device.pem` exists and is a valid certificate: `openssl x509 -in /opt/okta-verify/device.pem -noout -text` (or `openssl pkcs7 -print_certs -in /opt/okta-verify/device.pem` if Fleet returned a PKCS7 envelope).
3. In Okta, head to **Directory > Devices** and confirm the host appears with **Platform** Linux and **Enrolled By** Okta Verify.
4. On the host, open Okta Verify and confirm FastPass is available for sign-in.

## Troubleshoot

**`request_certificate` returns a 400 error**

The Request certificate API doesn't yet support Okta/NDES CAs — this is expected until [#52993](https://github.com/fleetdm/fleet/issues/52993) ships. Track that issue for availability.

**Okta Verify doesn't detect the certificate**

Okta hasn't published the exact path Okta Verify for Linux expects. Re-check `CERT_DIR` in the script against Okta Verify's actual installed layout on the host, and update it if needed.

<meta name="articleTitle" value="Configure Okta FastPass for Linux">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-14">
<meta name="description" value="Deploy Okta Verify and its FastPass device certificate to Linux hosts with Fleet.">
