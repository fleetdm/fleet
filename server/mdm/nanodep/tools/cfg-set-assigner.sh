#!/bin/sh

URL="${BASE_URL}/v1/assigner/${DEP_NAME}?profile_uuid=$1"

curl \
	$CURL_OPTS \
	-u "depserver:$APIKEY" \
	-X PUT \
	"$URL"
