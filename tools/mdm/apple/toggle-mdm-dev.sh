#!/bin/bash

# To toggle MDM, run `source toggle-mdm-dev`
# Requires the env at $FLEET_ENV_PATH to contain logic something like:

# if [[ $USE_MDM == "1" ]]; then

# # for MDM server
# export FLEET_MDM_APPLE_ENABLE=1
# export FLEET_MDM_APPLE_SCEP_CHALLENGE=scepchallenge
# MDM_PATH={PATH_TO_YOUR_MDM_RELATED_KEYS_AND_CERTS}
# export FLEET_MDM_APPLE_SCEP_CERT=$MDM_PATH"fleet-mdm-apple-scep.crt"
# export FLEET_MDM_APPLE_SCEP_KEY=$MDM_PATH"fleet-mdm-apple-scep.key"
# export FLEET_MDM_APPLE_BM_SERVER_TOKEN=$MDM_PATH"downloadtoken.p7m"
# export FLEET_MDM_APPLE_BM_CERT=$MDM_PATH"fleet-apple-mdm-bm-public-key.crt"
# export FLEET_MDM_APPLE_BM_KEY=$MDM_PATH"fleet-apple-mdm-bm-private.key"
# #below files are from the shared Fleet 1Password
# export FLEET_MDM_APPLE_APNS_CERT=$MDM_PATH"mdmcert.download.push.pem"
# export FLEET_MDM_APPLE_APNS_KEY=$MDM_PATH"mdmcert.download.push.key"
# else
# unset FLEET_MDM_APPLE_ENABLE
# unset FLEET_MDM_APPLE_SCEP_CHALLENGE
# unset FLEET_MDM_APPLE_SCEP_CERT
# unset FLEET_MDM_APPLE_SCEP_KEY
# unset FLEET_MDM_APPLE_BM_SERVER_TOKEN
# unset FLEET_MDM_APPLE_BM_CERT
# unset FLEET_MDM_APPLE_BM_KEY
# #below files are from the shared Fleet 1Password
# unset FLEET_MDM_APPLE_APNS_CERT
# unset FLEET_MDM_APPLE_APNS_KEY
# fi

if [[ $USE_MDM == "1" ]]; then
export USE_MDM=0
else
export USE_MDM=1
fi

source $FLEET_ENV_PATH
