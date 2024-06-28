#!/bin/sh

set -e 
AUTH_TOKEN_ARG=""
MIGRATE_PERCENTAGE_ARG=""
MIGRATE_UDIDS_ARG=""

if [ -z "${MDMPROXY_SERVER_ADDRESS}" ]; then
	MDMPROXY_SERVER_ADDRESS=":8080"
fi

if [ -n "${MDMPROXY_AUTH_TOKEN}" ]; then
	AUTH_TOKEN_ARG="-auth-token \"${MDMPROXY_AUTH_TOKEN:?}\""
fi

if [ -n "${MDMPROXY_MIGRATE_PERCENTAGE}" ]; then
	MIGRATE_PERCENTAGE_ARG="-migrate-percentage \"${MDMPROXY_MIGRATE_PERCENTAGE:?}\""
fi

if [ -n "${MDMPROXY_MIGRATE_UDIDS}" ]; then
	MIGRATE_UDIDS_ARG="-migrate-udids \"${MDMPROXY_MIGRATE_UDIDS:?}\""
fi

eval exec /usr/bin/mdmproxy \
	${AUTH_TOKEN_ARG} \
	-existing-hostname "${MDMPROXY_EXISTING_HOSTNAME:?}" \
	-existing-url "${MDMPROXY_EXISTING_URL:?}" \
	-fleet-url "${MDMPROXY_FLEET_URL:?}"  \
	${MIGRATE_PERCENTAGE_ARG} \
	${MIGRATE_UDIDS_ARG} \
	-server-address "${MDMPROXY_SERVER_ADDRESS:?}"
