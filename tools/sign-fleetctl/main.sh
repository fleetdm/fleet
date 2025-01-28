#!/usr/bin/env bash
set -eo pipefail

check_env_var() {
    if [[ -z "${!1}" ]]; then
        echo "Error: Environment variable $1 not set."
        exit 1
    fi
}

# check required environment variables
check_env_var "APPLE_APPLICATION_CERTIFICATE"
check_env_var "APPLE_APPLICATION_CERTIFICATE_PASSWORD"
check_env_var "APPLE_APP_STORE_CONNECT_KEY"
check_env_var "APPLE_APP_STORE_CONNECT_KEY_ID"
check_env_var "APPLE_APP_STORE_CONNECT_ISSUER_ID"
check_env_var "FLEETCTL_BINARY_PATH"

cleanup() {
    echo "Cleaning up..."
    rm -f certificate.p12
    rm -rf private_keys
    rm -f fleetctl.zip
}

# trap EXIT signal to call cleanup function
trap cleanup EXIT

echo "Signing binary..."
printf "%s" "$APPLE_APPLICATION_CERTIFICATE" | base64 --decode > certificate.p12
rcodesign sign --p12-file certificate.p12 \
               --p12-password "$APPLE_APPLICATION_CERTIFICATE_PASSWORD" \
               --for-notarization "$FLEETCTL_BINARY_PATH"

echo "Notarizing binary..."
mkdir -p private_keys
printf "%s" "$APPLE_APP_STORE_CONNECT_KEY" > "private_keys/AuthKey_$APPLE_APP_STORE_CONNECT_KEY_ID.p8"
zip fleetctl.zip "$FLEETCTL_BINARY_PATH"
rcodesign notary-submit \
          --api-issuer "$APPLE_APP_STORE_CONNECT_ISSUER_ID" \
          --api-key "$APPLE_APP_STORE_CONNECT_KEY_ID" \
          --wait --max-wait-seconds 300 fleetctl.zip

