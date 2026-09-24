#!/bin/bash
# Example script used in this guide: https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#any-est-enrollment-over-secure-transport-ca
set -e

# Load the end user information, IdP token and IdP client ID.
. /opt/company/userinfo

URL="<IdP-introspection-URL>"

# Generate the password-protected private key
openssl genpkey -algorithm RSA -out /opt/company/CustomerUserNetworkAccess.key -pkeyopt rsa_keygen_bits:2048 -aes256 -pass pass:${PASSWORD}

# Generate CSR signed with that private key. The CN can be changed and DNS attribute omitted if your EST configuration allows it.
openssl req -new -sha256 -key /opt/company/CustomerUserNetworkAccess.key -out CustomerUserNetworkAccess.csr -subj /CN=CustomerUserNetworkAccess:${USERNAME} -addext "subjectAltName=DNS:example.com, email:$USERNAME, otherName:msUPN;UTF8:$USERNAME" -passin pass:${PASSWORD}

# Escape CSR for request
CSR=$(sed 's/$/\\n/' CustomerUserNetworkAccess.csr | tr -d '\n')
REQUEST='{ "csr": "'"${CSR}"'", "idp_oauth_url":"'"${URL}"'", "idp_token": "'"${TOKEN}"'", "idp_client_id": "'"${CLIENT_ID}"'", "return_pem_certificate": true }'

curl 'https://<Fleet-server-URL>/api/latest/fleet/certificate_authorities/<EST-CA-ID>/request_certificate' \
  -X 'POST' \
  -H 'accept: application/json, text/plain, */*' \
  -H 'authorization: Bearer '"$FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN" \
  -H 'content-type: application/json' \
  --data-raw "${REQUEST}" -o response.json

jq -r .certificate response.json > /opt/company/certificate.pem
