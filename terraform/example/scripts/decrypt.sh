#!/bin/bash

set -e

function usage() {
	cat <<-EOUSAGE
	
	Usage: $(basename ${0}) <KMS_KEY_ID> <SOURCE> <DESTINATION> [AWS_PROFILE]
	
		This script decrypts an AWS KMS encrypted file from the desired
		SOURCE and places it it as the DESTINATION file.  Optionally you
		may provide the AWS_PROFILE you wish to use to run the aws kms
		commands.

		Hint: You can use /dev/stdout for the destination to just view the
		output.
	EOUSAGE
	exit 1
}

[ $# -lt 3 ] && usage

if [ -n "${4}" ]; then
	export AWS_PROFILE=${4}
fi

aws kms decrypt --key-id "${1:?}" --ciphertext-blob fileb://<(cat "${2:?}" | base64 -d) --output text --query Plaintext | base64 --decode > "${3:?}"
