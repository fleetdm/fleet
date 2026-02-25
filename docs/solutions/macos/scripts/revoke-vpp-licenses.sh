#!/bin/sh

# 1. Download the VPP token from Apple Business Manager (ABM).
#   a. In ABM, go to Account name in bottom left corner > Preferences > Payments and Billing > Download Content Token and download token for your location.
#   b. Open the downloaded token and copy base64. Paste base64 string instead of '{vpp_token}' in the curl command below.
# 2. Find `adamId` in the App Store app and use it in the assets array in the curl command below.
#   a. It can be retrieved from the app URL (e.g. 1487937127 from https://apps.apple.com/ba/app/craft-write-docs-ai-editing/id1487937127)
# 3. Add the serial numbers of the devices from which you want to revoke licenses to the `serialNumbers` array in the curl command below.
# Note: When a license is revoked, it takes some time for that to be reflected in Apple Business Manager.

curl -X POST https://vpp.itunes.apple.com/mdm/v2/assets/disassociate \
-H "Authorization: Bearer {vpp_token}" \
-H "Content-Type: application/json" \
-d '{
    "assets": [
        {
            "adamId": "1091189122",
            "pricingParam": "STDQ"
        }
    ],
    "serialNumbers": [
        "A641592ZDB"
    ]
}'

