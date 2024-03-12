#!/bin/sh

URL="${BASE_URL}/v1/tokenpki/${DEP_NAME}"

curl \
	$CURL_OPTS \
	-u "depserver:$APIKEY" \
	-T "$1" \
	"$URL"
