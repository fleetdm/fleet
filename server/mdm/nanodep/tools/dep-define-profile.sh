#!/bin/sh

# See https://developer.apple.com/documentation/devicemanagement/define_a_profile

DEP_ENDPOINT=/profile
URL="${BASE_URL}/proxy/${DEP_NAME}${DEP_ENDPOINT}"

curl \
	$CURL_OPTS \
	-u "depserver:$APIKEY" \
	-X POST \
	-H 'Content-type: application/json;charset=UTF8' \
	-T "$1" \
	-A "nanodep-tools/0" \
	"$URL"
