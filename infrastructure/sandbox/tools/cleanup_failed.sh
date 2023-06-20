#!/bin/bash

set -e
set -x

function is_in() {
	ITEM="${1}"
	LIST="${2}"
	for VALUE in ${LIST}; do
		if [ "${ITEM}" = "${VALUE}" ]; then
			return 0
		fi
	done
	return 1
}

pushd "$(dirname ${0})/../JITProvisioner/deprovisioner/deploy_terraform"

terraform init -backend-config=backend.conf

export TF_VAR_eks_cluster="sandbox-prod"
export TF_VAR_mysql_secret="arn:aws:secretsmanager:us-east-2:411315989055:secret:/fleet/database/password/mysql-boxer-QGmEeA"

terraform workspace select default

FAILED_EXECUTIONS="$(aws stepfunctions list-executions --state-machine-arn arn:aws:states:us-east-2:411315989055:stateMachine:sandbox-prod | jq -r '.executions[] | select(.status=="FAILED") | .name' | awk -F- '{ print $1 }')"

EXISTING_WORKSPACES="$(terraform workspace list | grep -v default | awk '{ print $1 }')"

TO_DELETE="$( (echo "${FAILED_EXECUTIONS:?}"; echo "${EXISTING_WORKSPACES:?}") | sort | uniq -d)"

set +x
echo "You must be connected to the VPN to continue."
echo "To Delete:           $(wc -l <<<"${TO_DELETE:?}")"
echo "Failed Executions:   $(wc -l <<<"${FAILED_EXECUTIONS:?}")"
echo "Existing Workspaces: $(wc -l <<<"${EXISTING_WORKSPACES}")"
echo "Press ENTER to continue, CTRL+C to abort"
read
set -x

for INSTANCE in ${TO_DELETE:?}; do
	if ! is_in "${INSTANCE}" "${EXISTING_WORKSPACES}"; then
		echo ${INSTANCE} is not in the existing workspaces, continuing.
		continue;
	fi
        terraform workspace select ${INSTANCE:?}
        echo "Destroying ${INSTANCE:?}"
        terraform apply -destroy -auto-approve
        terraform workspace select default
        echo "Deleting Workspace ${INSTANCE:?}"
        terraform workspace delete ${INSTANCE:?}
done

popd
