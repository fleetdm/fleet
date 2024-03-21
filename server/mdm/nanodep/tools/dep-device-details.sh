#!/bin/sh

# See https://developer.apple.com/documentation/devicemanagement/get_device_details

DEP_ENDPOINT=/devices
URL="${BASE_URL}/proxy/${DEP_NAME}${DEP_ENDPOINT}"

jq -n --arg device "$1" '.devices = [$device]' \
	| curl \
		$CURL_OPTS \
		-u "depserver:$APIKEY" \
		-X POST \
		-H 'Content-type: application/json;charset=UTF8' \
		--data-binary @- \
		-A "nanodep-tools/0" \
		"$URL"
