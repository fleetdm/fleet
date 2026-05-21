#!/bin/bash
set -euo pipefail

: "${ENROLL_SECRET:?ENROLL_SECRET must be set}"

awk -v s="$ENROLL_SECRET" '{gsub(/placeholder/, s)}1' \
	/etc/default/orbit > /etc/default/orbit.new \
	&& mv /etc/default/orbit.new /etc/default/orbit

set -a; . /etc/default/orbit; set +a

exec /opt/orbit/bin/orbit/orbit
