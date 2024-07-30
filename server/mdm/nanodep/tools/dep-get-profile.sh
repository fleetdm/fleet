#!/bin/sh

# See https://developer.apple.com/documentation/devicemanagement/get_a_profile

DEP_ENDPOINT=/profile
URL="${BASE_URL}/proxy/${DEP_NAME}${DEP_ENDPOINT}?profile_uuid=$1"

curl \
		$CURL_OPTS \
		-u "depserver:$APIKEY" \
		-A "nanodep-tools/0" \
		"$URL"
