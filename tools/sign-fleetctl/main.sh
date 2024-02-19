#!/usr/bin/env bash

echo "signing binary..."
sh -c 'echo "$APPLE_APPLICATION_CERTIFICATE" | base64 --decode > certificate.p12'
rcodesign sign --p12-file certificate.p12 \
       --p12-password "$APPLE_APPLICATION_CERTIFICATE_PASSWORD" \
       --for-notarization "$FLEETCTL_BINARY_PATH"
rm certificate.p12

echo "notarizing binary..."
mkdir private_keys
sh -c 'echo "$APPLE_APP_STORE_CONNECT_KEY" > "private_keys/AuthKey_$APPLE_APP_STORE_CONNECT_KEY_ID.p8"'
zip fletctl.zip "$FLEETCTL_BINARY_PATH"
rcodesign notary-submit \
       --api-issuer "$APPLE_APP_STORE_CONNECT_ISSUER_ID" \
       --api-key "$APPLE_APP_STORE_CONNECT_KEY_ID" \
       --wait fleetctl.zip
rm -rf private_keys
