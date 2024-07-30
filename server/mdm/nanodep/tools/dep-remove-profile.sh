#!/bin/sh

# See https://developer.apple.com/documentation/devicemanagement/remove_a_profile-c2c
# Note that while the docs contain a profile_uuid field it is not required.

DEP_ENDPOINT=/profile/devices
URL="${BASE_URL}/proxy/${DEP_NAME}${DEP_ENDPOINT}"

jq -n --arg device "$1" '.devices = [$device]' \
	| curl \
		$CURL_OPTS \
		-u "depserver:$APIKEY" \
		-X DELETE \
		-H 'Content-type: application/json;charset=UTF8' \
		--data-binary @- \
		-A "nanodep-tools/0" \
		"$URL"
