#!/bin/bash
# Script template used in this guide: https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#http-signatures
set -e

# Load the end user information, IdP token and IdP client ID.
. /opt/company/userinfo

URL="<IdP-introspection-URL>"

# Generate the password-protected private key
openssl genpkey -algorithm RSA -out /opt/company/CustomerUserNetworkAccess.key -pkeyopt rsa_keygen_bits:2048 -aes256 -pass pass:${PASSWORD}

# Generate CSR signed with that private key. The CN can be changed and DNS attribute omitted if your EST configuration allows it.
openssl req -new -sha256 -key /opt/company/CustomerUserNetworkAccess.key -out CustomerUserNetworkAccess.csr -subj /CN=CustomerUserNetworkAccess:${USERNAME} -addext "subjectAltName=DNS:example.com, email:$USERNAME, otherName:msUPN;UTF8:$USERNAME" -passin pass:${PASSWORD}

fetch_cert -ca <EST-CA-ID> -fleeturl "<Fleet-server-URL>" -csr CustomerUserNetworkAccess.csr -out /opt/company/certificate.pem
