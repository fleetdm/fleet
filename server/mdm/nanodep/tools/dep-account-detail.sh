#!/bin/sh

# See https://developer.apple.com/documentation/devicemanagement/get_account_detail

DEP_ENDPOINT=/account
URL="${BASE_URL}/proxy/${DEP_NAME}${DEP_ENDPOINT}"

curl \
	$CURL_OPTS \
	-u depserver:$APIKEY \
	-A "nanodep-tools/0" \
	"$URL"

