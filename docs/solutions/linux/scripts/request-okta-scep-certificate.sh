#!/bin/bash
# NOTE: This doesn't work yet. Fleet's "Request certificate" API endpoint doesn't
# support Okta's CA until https://github.com/fleetdm/fleet/issues/52993 ships.
# Learn more: https://fleetdm.com/guides/deploy-okta-fastpass-for-linux
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

# Fleet replaces $FLEET_VAR_NDES_SCEP_CHALLENGE with a one-time challenge from Okta.
openssl req -new -sha256 -key "$KEY_PATH" -out /tmp/okta-verify.csr -config <(cat <<EOF
[req]
prompt = no
distinguished_name = dn
attributes = attrs

[dn]
CN = $FLEET_VAR_HOST_END_USER_IDP_USERNAME Okta FastPass

[attrs]
challengePassword = $FLEET_VAR_NDES_SCEP_CHALLENGE
EOF
)

# Escape the CSR for the JSON request body.
CSR=$(sed 's/$/\\n/' /tmp/okta-verify.csr | tr -d '\n')
REQUEST='{ "csr": "'"${CSR}"'", "return_pem_certificate": true }'

curl "${FLEET_URL}/api/latest/fleet/certificate_authorities/${CA_ID}/request_certificate" \
  --fail --silent --show-error \
  -X 'POST' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'authorization: Bearer '"$FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN" \
  -H 'content-type: application/json' \
  --data-raw "${REQUEST}" -o /tmp/okta-verify-response.json

jq -r .certificate /tmp/okta-verify-response.json > "$CERT_PATH"
chmod 644 "$CERT_PATH"

rm -f /tmp/okta-verify.csr /tmp/okta-verify-response.json
