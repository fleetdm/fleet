#!/bin/bash

set -e

function usage() {
	cat <<-EOUSAGE
	
	Usage: $(basename ${0}) <KMS_KEY_ID> <SOURCE> <DESTINATION> [AWS_PROFILE]
	
		This script encrypts an plaintext file from SOURCE into an
		AWS KMS encrypted DESTINATION file.  Optionally you
		may provide the AWS_PROFILE you wish to use to run the aws kms
		commands.

	EOUSAGE
	exit 1
}

[ $# -lt 3 ] && usage

if [ -n "${4}" ]; then
	export AWS_PROFILE=${4}
fi

aws kms encrypt --key-id "${1:?}" --plaintext fileb://<(cat "${2:?}") --output text --query CiphertextBlob > "${3:?}"
